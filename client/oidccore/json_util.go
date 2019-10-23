package oidccore

import (
	"encoding/json"
	"net/url"
	"time"
)

type jsonStringOrStringList struct {
	Value []string
}

func (jo *jsonStringOrStringList) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var err error

	var list []string
	var single string

	if err = json.Unmarshal(data, &single); err == nil {
		*jo = jsonStringOrStringList{Value: []string{single}}
		return nil
	}

	if err = json.Unmarshal(data, &list); err == nil {
		*jo = jsonStringOrStringList{Value: list}
		return nil
	}

	return err
}

type jsonURL struct {
	Value *url.URL
}

func (jo *jsonURL) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	u, err := url.Parse(str)
	if err != nil {
		return err
	}
	*jo = jsonURL{Value: u}
	return nil
}

type jsonUnixTime struct {
	Value time.Time
}

func (jo *jsonUnixTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var seconds float64
	if err := json.Unmarshal(data, &seconds); err != nil {
		return err
	}
	*jo = jsonUnixTime{
		Value: time.Unix(0, 0).Add(time.Duration(seconds * float64(time.Second))),
	}
	return nil
}
