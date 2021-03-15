package pdutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pingcap/errors"

	"github.com/pingcap/tipocket/pkg/nemesis/fake_kvproto/metapb"
	httputil "github.com/pingcap/tipocket/pkg/util/http"
)

const (
	schedulersPrefix        = "/pd/api/v1/schedulers"
	regionKeyPrefix         = "/pd/api/v1/region/key/"
	regionsSiblingPrefix    = "/pd/api/v1/regions/sibling/"
	operatorsPrefix         = "/pd/api/v1/operators"
	storesPrefix            = "/pd/api/v1/stores"
	regionsPrefix           = "/pd/api/v1/regions"
	pdLeaderTransferPrefix  = "/pd/api/v1/leader/transfer"
	membersPrefix           = "/pd/api/v1/members"
	transferAllocatorPrefix = "/pd/api/v1/tso/allocator/transfer"
	storePrefix             = "/pd/api/v1/store"
	transferLeaderPrefix    = "/pd/api/v1/leader/transfer"

	contentJSON = "application/json"
)

// RegionInfo represents PD region info.
type RegionInfo struct {
	ID       uint64         `json:"id"`
	StartKey string         `json:"start_key"`
	EndKey   string         `json:"end_key"`
	Peers    []*metapb.Peer `json:"peers,omitempty"`
	Leader   *metapb.Peer   `json:"leader,omitempty"`
}

// Stores represents PD store response.
type Stores struct {
	Count  uint64       `json:"count"`
	Stores []*StoreInfo `json:"stores"`
}

// StoreInfo represents PD store info.
type StoreInfo struct {
	*metapb.Store `json:"store"`
}

// Client is a HTTP Client for PD.
type Client struct {
	c      *httputil.Client
	pdAddr string
}

// NewPDClient creates a HTTP Client for PD.
func NewPDClient(c *http.Client, pdAddr string) *Client {
	return &Client{c: httputil.NewHTTPClient(c), pdAddr: pdAddr}
}

// AddScheduler adds the specified scheduler to PD.
func (p *Client) AddScheduler(schedulerName string) error {
	input := map[string]string{"name": schedulerName}
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	_, err = p.c.Post(p.pdAddr+schedulersPrefix, "application/json", bytes.NewBuffer(data))
	return err
}

// RemoveScheduler removes the specified scheduler from PD.
func (p *Client) RemoveScheduler(schedulerName string) error {
	return p.c.Delete(p.pdAddr + schedulersPrefix + "/" + schedulerName)
}

// ListRegions lists region infos.
func (p *Client) ListRegions() ([]*RegionInfo, error) {
	resp, err := p.c.Get(p.pdAddr + regionsPrefix)
	if err != nil {
		return nil, err
	}
	var body struct {
		Regions []*RegionInfo `json:"regions"`
	}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal `[]RegionInfo` failed")
	}
	return body.Regions, nil
}

// GetStores gets PD stores information.
func (p *Client) GetStores() (*Stores, error) {
	resp, err := p.c.Get(p.pdAddr + storesPrefix)
	if err != nil {
		return nil, err
	}
	stores := &Stores{}
	err = json.Unmarshal(resp, stores)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal `Stores` failed")
	}
	return stores, nil
}

// GetRegionByKey gets the region info by region key.
func (p *Client) GetRegionByKey(key string) (*RegionInfo, error) {
	resp, err := p.c.Get(p.pdAddr + regionKeyPrefix + url.QueryEscape(key))
	if err != nil {
		return nil, err
	}
	region := &RegionInfo{}
	err = json.Unmarshal(resp, region)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal `RegionInfo` failed")
	}
	return region, nil
}

// Operators sends PD operators request.
func (p *Client) Operators(input map[string]interface{}) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	_, err = p.c.Post(p.pdAddr+operatorsPrefix, contentJSON, bytes.NewBuffer(body))
	return err
}

// GetSiblingRegions gets the siblings' region info.
func (p *Client) GetSiblingRegions(id uint64) ([]*RegionInfo, error) {
	resp, err := p.c.Get(p.pdAddr + regionsSiblingPrefix + strconv.FormatUint(id, 10))
	if err != nil {
		return nil, err
	}
	var body struct {
		Regions []*RegionInfo `json:"regions"`
	}
	if err = json.Unmarshal(resp, &body); err != nil {
		return nil, errors.Wrap(err, "Unmarshal `[]RegionInfo` failed")
	}
	return body.Regions, nil
}

// TransferPDLeader transfer pd leader
func (p *Client) TransferPDLeader(memberName string) error {
	_, err := p.c.Post(p.pdAddr+pdLeaderTransferPrefix+"/"+memberName, contentJSON, nil)
	return err
}

// GetMembers get PD members info
func (p *Client) GetMembers() (*MembersInfo, error) {
	resp, err := p.c.Get(p.pdAddr + membersPrefix)
	if err != nil {
		return nil, err
	}
	members := &MembersInfo{}
	err = json.Unmarshal(resp, members)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// TransferAllocator transfer tso allocator to the target pd member
func (p *Client) TransferAllocator(name, dclocation string) error {
	apiURL := fmt.Sprintf("%s%s/%s?dcLocation=%s", p.pdAddr, transferAllocatorPrefix, name, dclocation)
	_, err := p.c.Post(apiURL, contentJSON, nil)
	return err
}

// SetStoreLabels set store labels
func (p *Client) SetStoreLabels(storeID uint64, labels map[string]string) error {
	apiURL := fmt.Sprintf("%s%s/%v/label", p.pdAddr, storePrefix, storeID)
	data, err := json.Marshal(labels)
	if err != nil {
		return err
	}
	_, err = p.c.Post(apiURL, contentJSON, bytes.NewBuffer(data))
	return err
}

// MembersInfo is PD members info returned from PD RESTful interface
//type Members map[string][]*pdpb.Member
type MembersInfo struct {
	Members             []*Member          `json:"members,omitempty"`
	Leader              *Member            `json:"leader,omitempty"`
	EtcdLeader          *Member            `json:"etcd_leader,omitempty"`
	TsoAllocatorLeaders map[string]*Member `json:"tso_allocator_leaders,omitempty"`
}

// Member is the PD member info
type Member struct {
	// name is the name of the PD member.
	Name string `json:"name,omitempty"`
	// member_id is the unique id of the PD member.
	MemberID uint64 `json:"member_id,omitempty"`
	// dc_location is the dcLocation of the PD member
	DcLocation string `json:"dc_location,omitempty"`
}
