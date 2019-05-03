package main

import "github.com/concourse/concourse/atc"

type getRequest struct {
	Source  atc.Source  `json:"source"`
	Params  atc.Params  `json:"params,omitempty"`
	Version atc.Version `json:"version,omitempty"`
}

type putRequest struct {
	Source atc.Source `json:"source"`
	Params atc.Params `json:"params,omitempty"`
}
