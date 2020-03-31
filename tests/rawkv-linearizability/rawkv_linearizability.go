package rawkvlinearizability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	pd "github.com/pingcap/pd/client"
	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/rawkv"
)

const (
	keyStart        = 100
	keyNum          = 50
	valueStart      = 1
	valueNum        = 10000
	readProbability = 60
	writeProbaility = 30
)

type StateType map[uint16]uint16

type Config struct {
	keyStart        int
	keyNum          int
	valueStart      int
	valueNum        int
	ReadProbability int
	WriteProbaility int
}

type rawkvClient struct {
	r        *rand.Rand
	cli      *rawkv.Client
	pd       pd.Client
	startKey []byte
	endKey   []byte
}

func (c *rawkvClient) SetUp(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	c.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	clusterName := nodes[0].ClusterName
	ns := nodes[0].Namespace
	//pdAddrs := []string{"127.0.0.1:2379"}
	pdAddrs := []string{fmt.Sprintf("%s-pd.%s.svc:2379", clusterName, ns)}
	if len(pdAddrs) == 0 {
		return errors.New("No pd node found")
	}

	conf := config.Default()
	conf.Raw.MaxScanLimit = keyNum
	var err error
	c.cli, err = rawkv.NewClient(ctx, pdAddrs, conf)
	if err != nil {
		log.Fatalf("create tikv client error: %v", err)
	}
	c.startKey = []byte(strconv.Itoa(keyStart))
	c.endKey = []byte(strconv.Itoa(keyStart + keyNum))
	// TODO: maybe we can insert some data before beginning test
	log.Printf("setup rawkv-linearizability")
	return nil
}

func (c *rawkvClient) TearDown(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	return nil
}

func (c *rawkvClient) Invoke(ctx context.Context, node clusterTypes.ClientNode, r interface{}) interface{} {
	request := r.(rawkvRequest)

	key := []byte(strconv.Itoa(int(request.Key)))
	switch request.Op {
	case 0:
		// get
		val, err := c.cli.Get(ctx, key)
		if err != nil {
			return rawkvResponse{Unknown: true, Error: err.Error()}
		}
		valueInt, _ := strconv.Atoi(string(val))
		return rawkvResponse{Value: uint16(valueInt)}
	case 1:
		// put
		value := []byte(strconv.Itoa(int(request.Value)))
		err := c.cli.Put(ctx, key, value)
		if err != nil {
			return rawkvResponse{Unknown: true, Error: err.Error()}
		}
		return rawkvResponse{}
	case 2:
		// delete
		err := c.cli.Delete(ctx, key)
		if err != nil {
			return rawkvResponse{Unknown: true, Error: err.Error()}
		}
		return rawkvResponse{}
	default:
		panic("unreachable")
	}
}

func (c *rawkvClient) NextRequest() interface{} {
	request := rawkvRequest{}
	rNum := uint16(c.r.Int() % 100)
	request.Op = 0
	request.Key = uint16(keyStart + c.r.Int()%keyNum)
	if rNum >= readProbability && rNum < readProbability+writeProbaility {
		// put
		request.Op = 1
		request.Value = uint16(valueStart + c.r.Int()%valueNum)
	} else if rNum >= readProbability+writeProbaility {
		// delete
		request.Op = 2
	}

	return request
}

func (c *rawkvClient) DumpState(ctx context.Context) (interface{}, error) {
	keys, values, err := c.cli.Scan(ctx, c.startKey, c.endKey, keyNum)
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}
	state := make(StateType)
	for i, key := range keys {
		value := values[i]
		keyInt, _ := strconv.Atoi(string(key))
		valueInt, _ := strconv.Atoi(string(value))
		state[uint16(keyInt)] = uint16(valueInt)
	}
	return state, nil
}

func (c *rawkvClient) Start(ctx context.Context, cfg interface{}, clientNodes []clusterTypes.ClientNode) error {
	return nil
}

type RawkvClientCreator struct{}

func (RawkvClientCreator) Create(node clusterTypes.ClientNode) core.Client {
	return &rawkvClient{}
}

// Request && Response
type rawkvRequest struct {
	Op    int
	Key   uint16
	Value uint16
}

type rawkvResponse struct {
	Value   uint16
	Unknown bool
	Error   string `json:",omitempty"`
}

func (t *rawkvResponse) IsUnknown() bool {
	return t.Unknown
}

type rawkvParser struct{}

func RawkvParser() history.RecordParser {
	return rawkvParser{}
}

func (rawkvParser) OnRequest(data json.RawMessage) (interface{}, error) {
	request := rawkvRequest{}
	err := json.Unmarshal(data, &request)
	return request, err
}

func (rawkvParser) OnResponse(data json.RawMessage) (interface{}, error) {
	response := rawkvResponse{}
	err := json.Unmarshal(data, &response)
	return response, err
}

func (rawkvParser) OnNoopResponse() interface{} {
	return rawkvResponse{Unknown: true}
}

func (rawkvParser) OnState(state json.RawMessage) (interface{}, error) {
	st := make(StateType)
	err := json.Unmarshal(state, &st)
	return st, err
}

// Model implementation
type rawkvModel struct {
	preparedState *StateType
}

func RawkvModel() core.Model {
	return &rawkvModel{}
}

func (m *rawkvModel) Prepare(state interface{}) {
	s := state.(StateType)
	m.preparedState = &s
}

func (m *rawkvModel) Init() interface{} {
	if m.preparedState != nil {
		return *m.preparedState
	}

	return make(StateType)
}

func (m *rawkvModel) Step(state interface{}, input interface{}, output interface{}) (bool, interface{}) {
	st := state.(StateType)
	request := input.(rawkvRequest)
	response := output.(rawkvResponse)

	switch request.Op {
	case 0:
		// get
		if response.Unknown {
			return true, state
		}
		value, ok := st[request.Key]
		if !ok {
			// empty
			value = 0
		}
		if value == response.Value {
			return true, state
		}
		return false, state
	case 1, 2:
		// put or delete
		// we can safely assume the put/delete operation is succeed because all of
		// the unknown reponses are moved to the end of events.
		newSt := make(StateType)
		// TODO: we should use **Persistent Binary Search Tree** here and it
		// can save more memory when there are many states.
		// Copy from the original map to the target map
		for key, value := range st {
			newSt[key] = value
		}
		if request.Op == 1 {
			newSt[request.Key] = request.Value
		} else {
			delete(newSt, request.Key)
		}
		return true, newSt
	default:
		panic("unreachable")
	}
}

func (m *rawkvModel) Equal(state1, state2 interface{}) bool {
	st1 := state1.(StateType)
	st2 := state2.(StateType)

	// Maybe we can use hash to speed up
	if len(st1) != len(st2) {
		return false
	}

	for key, value := range st1 {
		value2, ok := st2[key]
		if !ok || value != value2 {
			return false
		}
	}
	return true
}

func (*rawkvModel) Name() string {
	return "rawkv-linearizability"
}
