package control

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

// LeakCheck ...
type LeakCheck struct {
	*Controller
}

// InitPlugin ...
func (l *LeakCheck) InitPlugin(control *Controller) {
	l.Controller = control
	if len(fixture.Context.LeakCheckEatFile) == 0 {
		return
	}
	if err := template(); err != nil {
		log.Warnf("plugin leak check won't work: %v", err)
		return
	}
	var tidbNodes []types.Node
	for _, node := range l.Controller.cfg.Nodes {
		if node.Component == types.TiDB {
			tidbNodes = append(tidbNodes, node)
		}
	}
	go func() {
		log.Info("leak check is running...")
		for {
			select {
			case <-l.ctx.Done():
				log.Info("leak check is finished")
				return
			default:
				if err := checkTiDBProcess(tidbNodes); err != nil {
					log.Warnf("leak check occurred an error: %+v", err)
				}
				time.Sleep(30 * time.Second)
			}
		}
	}()
}

func checkTiDBProcess(tidbNodes []types.Node) error {
	for _, tidbNode := range tidbNodes {
		// tidbAddr is host:4000, but we need host:10080
		tidbAddr := fmt.Sprintf("%s.%s-tidb-peer.%s.svc", tidbNode.PodName, tidbNode.ClusterName, tidbNode.ClusterName)
		stacks, body, err := goroutineProfile(tidbAddr)
		if err != nil {
			return errors.Trace(err)
		}
		if err := checkLeak(stacks, body, tidbAddr); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

var leagel map[string]*Stack

func checkLeak(stacks []*Stack, body, tidbIP string) error {
	var leak bool
	var buf bytes.Buffer
	for _, s := range stacks {
		// Consider goroutine lived longer than 20 minutes.
		if s.Up < 20 {
			continue
		}

		sig := s.Signature()
		if _, ok := leagel[string(sig)]; !ok {
			buf.WriteString("goroutine seems to leak:\n")
			buf.Write(s.Raw)
			fmt.Fprintf(&buf, "\n--------%s %s", tidbIP, time.Now())
			leak = true
		}
	}
	if leak {
		log.Error(body)
		log.Fatal(buf.String())
	}
	log.Infof("leak check successfully: %s", tidbIP)
	return nil
}

func template() error {
	stacks, err := parseFile()
	if err != nil {
		return errors.Trace(err)
	}
	leagel = make(map[string]*Stack)

	for i := 0; i < len(stacks); i++ {
		s := stacks[i]
		if s.Up > 0 {
			sig := s.Signature()
			leagel[string(sig)] = s
		}
	}
	return nil
}

func goroutineProfile(host string) ([]*Stack, string, error) {
	url := fmt.Sprintf("http://%s:10080/debug/pprof/goroutine?debug=2", host)
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", errors.Trace(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.Trace(err)
	}
	stack, err := parseData(body)
	return stack, string(body), err
}

func splitText(input []byte) [][]byte {
	return bytes.Split(input, []byte("\n\n"))
}

// Stack is stack
type Stack struct {
	ID    uint64
	State string
	Up    int
	Func  []string
	Line  []string
	Raw   []byte
}

// Signature signs stack
func (s *Stack) Signature() []byte {
	h := md5.New()
	for _, f := range s.Func {
		pos := strings.LastIndexByte(f, '(')
		if pos > 0 {
			io.WriteString(h, f[:pos])
		} else {
			io.WriteString(h, f)
		}
	}
	return h.Sum(nil)
}

func parseFile() ([]*Stack, error) {
	resp, err := http.Get(fixture.Context.LeakCheckEatFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()

	text, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return parseData(text)
}

func parseData(text []byte) ([]*Stack, error) {
	texts := splitText(text)
	ret := make([]*Stack, 0, len(texts))
	for _, text := range texts {
		s, err := parseStack(text)
		if err != nil {
			return nil, errors.Trace(err)
		}
		ret = append(ret, s)
	}
	return ret, nil
}

func parseStack(text []byte) (*Stack, error) {
	var stack Stack
	lines := bytes.Split(text, []byte{'\n'})
	if len(lines) < 3 {
		return nil, errors.Errorf("wrong input:\n%s", text)
	}
	if err := parseGoroutineLine(string(lines[0]), &stack); err != nil {
		return nil, errors.Trace(err)
	}

	lines = lines[1:]
	stack.Func = make([]string, 0, len(lines)/2)
	stack.Line = make([]string, 0, len(lines)/2)
	for len(lines) >= 2 {
		stack.Func = append(stack.Func, string(lines[0]))
		stack.Line = append(stack.Line, string(lines[1]))
		lines = lines[2:]
	}
	stack.Raw = text
	return &stack, nil
}

func parseGoroutineLine(text string, s *Stack) error {
	// text looks like that "goroutine 124 [select, 72 minutes]:"
	fmt.Sscanf(text, "goroutine %d", &s.ID)
	pos := strings.IndexByte(text, '[')
	if pos < 0 {
		return errors.Errorf("wrong input:%s", text)
	}
	text = text[pos+1 : len(text)-2]
	texts := strings.Split(text, ", ")
	s.State = texts[0]
	if len(texts) > 1 && strings.Contains(texts[1], "minutes") {
		fmt.Sscanf(texts[1], "%d minutes", &s.Up)
	}
	return nil
}
