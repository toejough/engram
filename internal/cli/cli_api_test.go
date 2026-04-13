package cli_test

import "engram/internal/apiclient"

//go:generate impgen apiclient.API --dependency --import-path engram/internal/apiclient

// Type aliases so the generated MockAPI (which uses unqualified names) compiles
// in the cli_test package.
type (
	API                 = apiclient.API
	PostMessageRequest  = apiclient.PostMessageRequest
	PostMessageResponse = apiclient.PostMessageResponse
	WaitRequest         = apiclient.WaitRequest
	WaitResponse        = apiclient.WaitResponse
	SubscribeRequest    = apiclient.SubscribeRequest
	SubscribeResponse   = apiclient.SubscribeResponse
	StatusResponse      = apiclient.StatusResponse
)
