package subs

import "github.com/gnoswap-labs/grc20-register/serve/conns"

type getWSConnectionDelegate func(string) conns.WSConnection

type mockConnectionFetcher struct {
	getWSConnectionFn getWSConnectionDelegate
}

func (m *mockConnectionFetcher) GetWSConnection(id string) conns.WSConnection {
	if m.getWSConnectionFn != nil {
		return m.getWSConnectionFn(id)
	}

	return nil
}
