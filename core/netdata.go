package core

import (
	"encoding/json"
	"io"
)

type Reqeust struct {
	Action string `json:"action"`
}

type Response struct {
	Message string `json:"message"`
}

type BindRequest struct {
	Reqeust
	ClientName string `json:"clientName"`
	OpenPort   string `json:"openPort"`
}

type BindResponse struct {
	Response
	ClientName   string `json:"clientName"`
	RelayPort    int    `json:"relayPort"`
	HandshakeKey string `json:"handshakeKey"`
}

type UnBindRequest struct {
	ClientName string `json:"clientName"`
}

func ReadJson2Object(r io.Reader, obj any) error {
	buff := make([]byte, 0, 1024)
	tmp := make([]byte, 1)
	count := 0
	for {
		size, err := r.Read(tmp)
		if err != nil {
			return err
		}
		buff = append(buff, tmp[:size]...)
		count += size
		if tmp[0] == '\n' {
			break
		}
	}
	return json.Unmarshal(buff[:count], obj)
}

func WriteObject2Json(w io.Writer, obj any) error {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = w.Write(append(jsonData, '\n'))
	return err
}
