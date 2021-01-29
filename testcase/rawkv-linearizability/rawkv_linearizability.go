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

	persistent_treap "github.com/gengliqi/persistent_treap/persistent_treap"
	pd "github.com/pingcap/pd/client"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/rawkv"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
	"github.com/pingcap/tipocket/util"
)

// Key type
type Key int

// Value type
type Value uint32

// KeyValuePair type
type KeyValuePair struct {
	K Key
	V Value
}

// Equals implements key's persistent_treap.Equitable interface
func (a Key) Equals(b persistent_treap.Equitable) bool {
	return a == b.(Key)
}

// Less implements persistent_treap.Sortable interface
func (a Key) Less(b persistent_treap.Sortable) bool {
	return a < b.(Key)
}

// Equals implements persistent_treap.Equitable interface
func (a Value) Equals(b persistent_treap.Equitable) bool {
	return a == b.(Value)
}

type stateType struct {
	treap persistent_treap.PersistentTreap
	hash  uint64
}

func newState() stateType {
	return stateType{persistent_treap.NewPersistentTreap(), 0}
}

var magicNumberKey = uint64(6364136223846793005)
var magicNumberValue = uint64(1103515245)

func (s stateType) Insert(k Key, v Value) stateType {
	newState := stateType{}
	val, ok := s.treap.GetValue(k)
	if ok {
		newState.hash = s.hash + (uint64(v)-uint64(val.(Value)))*magicNumberValue
	} else {
		newState.hash = s.hash + uint64(k)*magicNumberKey + uint64(v)*magicNumberValue + 12345
	}
	newState.treap = s.treap.Insert(k, v)
	return newState
}

func (s stateType) Remove(k Key) stateType {
	val, ok := s.treap.GetValue(k)
	if ok {
		newState := stateType{}
		newState.hash = s.hash - (uint64(k)*magicNumberKey + uint64(val.(Value))*magicNumberValue + 12345)
		newState.treap = s.treap.Remove(k)
		return newState
	}
	return s
}

// Config is the config of the test case.
type Config struct {
	KeyStart        int
	KeyNum          int
	ReadProbability int
	WriteProbaility int
}

// RandomValues is some random byte slices which have different hash value.
type RandomValues struct {
	hashs        []uint32
	hashValueMap map[uint32][]byte
}

// RandomValueConfig is the config of generating RandomValues
type RandomValueConfig struct {
	ValueNum10KB  int
	ValueNum100KB int
	ValueNum1MB   int
	ValueNum5MB   int
}

// GenerateRandomValueString generates RandomValues
func GenerateRandomValueString(config RandomValueConfig) RandomValues {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	r := RandomValues{
		hashValueMap: make(map[uint32][]byte),
	}
	value1KB := 1000
	type numToLen struct {
		num int
		len int
	}
	valueNum := []numToLen{
		{config.ValueNum10KB, 10 * value1KB},
		{config.ValueNum100KB, 100 * value1KB},
		{config.ValueNum1MB, 1024 * value1KB},
		{config.ValueNum5MB, 5120 * value1KB},
	}
	for i := 0; i < len(valueNum); i++ {
		for j := 0; j < valueNum[i].num; j++ {
			str := make([]byte, valueNum[i].len)
			for {
				util.RandString(str, rnd)
				h32 := util.Hashfnv32a(str)
				if _, ok := r.hashValueMap[h32]; !ok {
					r.hashs = append(r.hashs, h32)
					r.hashValueMap[h32] = str
					break
				}
			}
		}
	}
	return r
}

// RawkvClientCreator creates a test client.
type RawkvClientCreator struct {
	Cfg          Config
	RandomValues *RandomValues
}

type rawkvClient struct {
	r            *rand.Rand
	cli          *rawkv.Client
	pd           pd.Client
	conf         Config
	randomValues *RandomValues
}

// SetUp implements the core.Client interface.
func (c *rawkvClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	log.Printf("setup client %v start", idx)

	c.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	clusterName := clientNodes[0].ClusterName
	ns := clientNodes[0].Namespace
	pdAddrs := []string{fmt.Sprintf("%s-pd.%s.svc:2379", clusterName, ns)}
	//pdAddrs := []string{"127.0.0.1:2379"}
	if len(pdAddrs) == 0 {
		return errors.New("No pd node found")
	}

	var err error
	c.cli, err = rawkv.NewClient(ctx, pdAddrs, config.Default())
	if err != nil {
		log.Fatalf("create tikv client error: %v", err)
	}

	if idx == 0 {
		for i := c.conf.KeyStart; i < c.conf.KeyStart+c.conf.KeyNum; i++ {
			key := []byte(strconv.Itoa(i))
			err := c.cli.Delete(ctx, key)
			if err != nil {
				log.Fatalf("delete key %v error: %v", i, err)
			}
		}
		log.Println("client 0 delete all related key value")
	}

	log.Printf("setup client %v end", idx)

	return nil
}

// TearDown implements the core.Client interface.
func (c *rawkvClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *rawkvClient) ScheduledClientExtensions() core.OnScheduleClientExtensions {
	return c
}

func (c *rawkvClient) StandardClientExtensions() core.StandardClientExtensions {
	return nil
}

// Invoke implements the core.Client interface.
func (c *rawkvClient) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) core.UnknownResponse {
	request := r.(rawkvRequest)
	key := []byte(strconv.Itoa(request.Key))
	switch request.Op {
	case 0:
		// get
		val, err := c.cli.Get(ctx, key)
		if err != nil {
			return rawkvResponse{Unknown: true, Error: err.Error()}
		}
		if val == nil {
			return rawkvResponse{Val: 0}
		}
		h64 := util.Hashfnv32a(val)
		if _, ok := c.randomValues.hashValueMap[h64]; !ok {
			log.Fatalf("value not valid! key %s, value hash %v", key, h64)
		}
		return rawkvResponse{Val: h64}
	case 1:
		// put
		err := c.cli.Put(ctx, key, c.randomValues.hashValueMap[request.Val])
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

// NextRequest implements the core.Client interface.
func (c *rawkvClient) NextRequest() interface{} {
	request := rawkvRequest{}
	rNum := c.r.Int() % 100
	request.Op = 0
	request.Key = c.conf.KeyStart + c.r.Int()%c.conf.KeyNum
	if rNum >= c.conf.ReadProbability && rNum < c.conf.ReadProbability+c.conf.WriteProbaility {
		// put
		request.Op = 1
		request.Val = c.randomValues.hashs[int(c.r.Int31())%len(c.randomValues.hashs)]
	} else if rNum >= c.conf.ReadProbability+c.conf.WriteProbaility {
		// delete
		request.Op = 2
	}

	return request
}

// DumpState implements the core.Client interface.
func (c *rawkvClient) DumpState(ctx context.Context) (interface{}, error) {
	var kvs []KeyValuePair
	for i := c.conf.KeyStart; i < c.conf.KeyStart+c.conf.KeyNum; i++ {
		key := []byte(strconv.Itoa(i))
		for {
			val, err := c.cli.Get(ctx, key)
			if err == nil {
				if val != nil {
					h32 := util.Hashfnv32a(val)
					kvs = append(kvs, KeyValuePair{
						K: Key(i),
						V: Value(h32),
					})
				}
				break
			}
		}
	}
	log.Printf("dumpstate kv num %v", len(kvs))
	return kvs, nil
}

// Create creates a RawkvClient.
func (r RawkvClientCreator) Create(node cluster.ClientNode) core.Client {
	return &rawkvClient{
		conf:         r.Cfg,
		randomValues: r.RandomValues,
	}
}

// Request && Response
type rawkvRequest struct {
	Op  int
	Key int
	Val uint32
}

type rawkvResponse struct {
	Val     uint32
	Unknown bool
	Error   string `json:",omitempty"`
}

// IsUnknown implements UnknownResponse interface
func (r rawkvResponse) IsUnknown() bool {
	return r.Unknown
}

// Parser implements the core.Parser interface.
type rawkvParser struct{}

// RawkvParser creates the history.RecordParser interface.
func RawkvParser() history.RecordParser {
	return rawkvParser{}
}

// OnRequest implements the core.Parser interface.
func (rawkvParser) OnRequest(data json.RawMessage) (interface{}, error) {
	request := rawkvRequest{}
	err := json.Unmarshal(data, &request)
	return request, err
}

// OnResponse implements the core.Parser interface.
func (rawkvParser) OnResponse(data json.RawMessage) (interface{}, error) {
	response := rawkvResponse{}
	err := json.Unmarshal(data, &response)
	return response, err
}

// OnNoopResponse implements the core.Parser interface.
func (rawkvParser) OnNoopResponse() interface{} {
	return rawkvResponse{Unknown: true}
}

// OnState implements the core.Parser interface.
func (rawkvParser) OnState(state json.RawMessage) (interface{}, error) {
	var dump []KeyValuePair
	err := json.Unmarshal(state, &dump)
	st := newState()
	for i := 0; i < len(dump); i++ {
		st = st.Insert(Key(dump[i].K), Value(dump[i].V))
	}
	return st, err
}

// Model implementation
type rawkvModel struct {
	preparedState *stateType
}

// RawkvModel creates the core.Model interface.
func RawkvModel() core.Model {
	return &rawkvModel{}
}

// Prepare implements the core.Model interface.
func (m *rawkvModel) Prepare(state interface{}) {
	s := state.(stateType)
	m.preparedState = &s
}

// Init implements the core.Model interface.
func (m *rawkvModel) Init() interface{} {
	if m.preparedState != nil {
		return *m.preparedState
	}
	return newState()
}

// Step implements the core.Model interface.
func (m *rawkvModel) Step(state interface{}, input interface{}, output interface{}) (bool, interface{}) {
	st := state.(stateType)
	request := input.(rawkvRequest)
	response := output.(rawkvResponse)
	switch request.Op {
	case 0:
		// get
		if response.Unknown {
			return true, state
		}
		val, ok := st.treap.GetValue(Key(request.Key))
		if !ok {
			// empty
			val = Value(0)
		}
		if val.(Value).Equals(Value(response.Val)) {
			return true, state
		}
		return false, state
	case 1:
		// we can safely assume the put/delete operation is succeed because all of
		// the unknown reponses are moved to the end of events.
		return true, st.Insert(Key(request.Key), Value(request.Val))
	case 2:
		return true, st.Remove(Key(request.Key))
	default:
		panic("unreachable")
	}
}

// Equal implements the core.Model interface.
func (m *rawkvModel) Equal(state1, state2 interface{}) bool {
	st1 := state1.(stateType)
	st2 := state2.(stateType)

	if st1 == st2 {
		return true
	}
	if st1.hash != st2.hash {
		return false
	}

	return persistent_treap.IsSameTreap(st1.treap, st2.treap)
}

// Name implements the core.Model interface.
func (*rawkvModel) Name() string {
	return "rawkv-linearizability"
}
