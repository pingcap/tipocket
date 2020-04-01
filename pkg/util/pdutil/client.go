package pdutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/metapb"

	httputil "github.com/pingcap/tipocket/pkg/util/http"
)

const (
	schedulersPrefix     = "/pd/api/v1/schedulers"
	regionKeyPrefix      = "/pd/api/v1/region/key/"
	regionsSiblingPrefix = "/pd/api/v1/regions/sibling/"
	operatorsPrefix      = "/pd/api/v1/operators"
	storesPrefix         = "/pd/api/v1/stores"
	regionsPrefix        = "/pd/api/v1/regions"

	contentJSON = "application/json"
)

// RegionInfo represents PDConfig region info.
type RegionInfo struct {
	ID       uint64         `json:"id"`
	StartKey string         `json:"start_key"`
	EndKey   string         `json:"end_key"`
	Peers    []*metapb.Peer `json:"peers,omitempty"`
	Leader   *metapb.Peer   `json:"leader,omitempty"`
}

// Stores represents PDConfig store response.
type Stores struct {
	Count  uint64       `json:"count"`
	Stores []*StoreInfo `json:"stores"`
}

// StoreInfo represents PDConfig store info.
type StoreInfo struct {
	*metapb.Store `json:"store"`
}

// Client is a HTTP Client for PDConfig.
type Client struct {
	c      *httputil.Client
	pdAddr string
}

// NewPDClient creates a HTTP Client for PDConfig.
func NewPDClient(c *http.Client, pdAddr string) *Client {
	return &Client{c: httputil.NewHTTPClient(c), pdAddr: pdAddr}
}

// AddScheduler adds the specified scheduler to PDConfig.
func (p *Client) AddScheduler(schedulerName string) error {
	input := map[string]string{"name": schedulerName}
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	if err := p.c.Post(p.pdAddr+schedulersPrefix, "application/json", bytes.NewBuffer(data)); err != nil {
		return err
	}
	return nil
}

// RemoveScheduler removes the specified scheduler from PDConfig.
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

// GetStores gets PDConfig stores information.
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

// Operators sends PDConfig operators request.
func (p *Client) Operators(input map[string]interface{}) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return p.c.Post(p.pdAddr+operatorsPrefix, contentJSON, bytes.NewBuffer(body))
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
