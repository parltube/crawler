package proxy

import (
	tls_client "github.com/bogdanfinn/tls-client"
)

type IRevolvingProxy interface {
	Next() (tls_client.HttpClient, int)
	Change(index int) error
}
