package model

import "encoding/json"

func UnmarshalSiteList(data []byte) (SiteList, error) {
	var r SiteList
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *SiteList) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type SiteList struct {
	FavoriteSiteIDS      []string `json:"favoriteSiteIds"`
	AutoFavoritesEnabled bool     `json:"autoFavoritesEnabled"`
}
