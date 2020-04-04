package rawkvlinearizability

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"strconv"
	"time"

	pd "github.com/pingcap/pd/client"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/rawkv"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"

	persistent_treap "github.com/gengliqi/persistent_treap/persistent_treap"
)

type Key int
type Value int

func (a Key) Equals(b persistent_treap.Equitable) bool {
	return a == b.(Key)
}

func (a Key) Less(b persistent_treap.Sortable) bool {
	return a < b.(Key)
}

func (a Value) Equals(b persistent_treap.Equitable) bool {
	return a == b.(Value)
}

type StateType struct {
	treap persistent_treap.PersistentTreap
	hash  uint64
}

func NewState() StateType {
	return StateType{persistent_treap.NewPersistentTreap(), 0}
}

var magicNumberKey = uint64(6364136223846793005)
var magicNumberValue = uint64(1103515245)

func (s StateType) Insert(k Key, v Value) StateType {
	newState := StateType{}
	val, ok := s.treap.GetValue(k)
	if ok {
		newState.hash = s.hash + (uint64(v)-uint64(val.(Value)))*magicNumberValue
	} else {
		newState.hash = s.hash + uint64(k)*magicNumberKey + uint64(v)*magicNumberValue + 12345
	}
	newState.treap = s.treap.Insert(k, v)
	return newState
}

func (s StateType) Remove(k Key) StateType {
	val, ok := s.treap.GetValue(k)
	if ok {
		newState := StateType{}
		newState.hash = s.hash - (uint64(k)*magicNumberKey + uint64(val.(Value))*magicNumberValue + 12345)
		newState.treap = s.treap.Remove(k)
		return newState
	}
	return s
}

type Config struct {
	KeyStart        int
	KeyNum          int
	ReadProbability int
	WriteProbaility int
}

type rawkvClient struct {
	r        *rand.Rand
	cli      *rawkv.Client
	pd       pd.Client
	conf     Config
	startKey []byte
	endKey   []byte
}

func (c *rawkvClient) SetUp(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	c.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	//clusterName := nodes[0].ClusterName
	//ns := nodes[0].Namespace
	//pdAddrs := []string{fmt.Sprintf("%s-pd.%s.svc:2379", clusterName, ns)}
	pdAddrs := []string{"127.0.0.1:2379"}
	if len(pdAddrs) == 0 {
		return errors.New("No pd node found")
	}

	conf := config.Default()
	conf.Raw.MaxScanLimit = c.conf.KeyNum
	var err error
	c.cli, err = rawkv.NewClient(ctx, pdAddrs, conf)
	if err != nil {
		log.Fatalf("create tikv client error: %v", err)
	}

	c.startKey = []byte(strconv.Itoa(c.conf.KeyStart))
	c.endKey = []byte(strconv.Itoa(c.conf.KeyStart + c.conf.KeyNum))

	log.Printf("startKey:%s endKey:%s", c.startKey, c.endKey)
	// TODO: maybe we can insert some data before beginning test
	log.Printf("setup rawkv-linearizability")
	return nil
}

func (c *rawkvClient) TearDown(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	return nil
}

func (c *rawkvClient) Invoke(ctx context.Context, node clusterTypes.ClientNode, r interface{}) core.UnknownResponse {
	request := r.(rawkvRequest)
	key := []byte(strconv.Itoa(request.Key))
	switch request.Op {
	case 0:
		// get
		val, err := c.cli.Get(ctx, key)
		if err != nil {
			return rawkvResponse{Unknown: true, Error: err.Error()}
		}
		valueInt, _ := strconv.Atoi(string(val))
		return rawkvResponse{Val: valueInt}
	case 1:
		// put
		value := []byte(strconv.Itoa(request.Val))
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
	rNum := c.r.Int() % 100
	request.Op = 0
	request.Key = c.conf.KeyStart + c.r.Int()%c.conf.KeyNum
	if rNum >= c.conf.ReadProbability && rNum < c.conf.ReadProbability+c.conf.WriteProbaility {
		// put
		request.Op = 1
		request.Val = int(c.r.Int31())
	} else if rNum >= c.conf.ReadProbability+c.conf.WriteProbaility {
		// delete
		request.Op = 2
	}

	return request
}

func (c *rawkvClient) DumpState(ctx context.Context) (interface{}, error) {
	var kvs [][2]int
	for i := c.conf.KeyStart; i < c.conf.KeyStart+c.conf.KeyNum; i++ {
		key := []byte(strconv.Itoa(i))
		val, err := c.cli.Get(ctx, key)
		valueInt, _ := strconv.Atoi(string(val))
		if err == nil {
			kvs = append(kvs, [2]int{i, valueInt})
		}
	}
	log.Printf("key num: %v between key %s and %s", len(kvs), c.startKey, c.endKey)
	return kvs, nil
}

func (c *rawkvClient) Start(ctx context.Context, cfg interface{}, clientNodes []clusterTypes.ClientNode) error {
	return nil
}

type RawkvClientCreator struct {
	Cfg Config
}

func (r RawkvClientCreator) Create(node clusterTypes.ClientNode) core.Client {
	return &rawkvClient{conf: r.Cfg}
}

// Request && Response
type rawkvRequest struct {
	Op  int
	Key int
	Val int
}

type rawkvResponse struct {
	Val     int
	Unknown bool
	Error   string `json:",omitempty"`
}

func (r rawkvResponse) IsUnknown() bool {
	return r.Unknown
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
	var dump [][2]int
	err := json.Unmarshal(state, &dump)
	st := NewState()
	for i := 0; i < len(dump); i++ {
		st = st.Insert(Key(dump[i][0]), Value(dump[i][1]))
	}
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

	return NewState()
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

func (m *rawkvModel) Equal(state1, state2 interface{}) bool {
	st1 := state1.(StateType)
	st2 := state2.(StateType)

	if st1 == st2 {
		return true
	}
	if st1.hash != st2.hash {
		return false
	}

	return persistent_treap.IsSameTreap(st1.treap, st2.treap)
}

func (*rawkvModel) Name() string {
	return "rawkv-linearizability"
}
