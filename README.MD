# testserver

testserver is a simple framework for mocking http server. It integrates with Go's built-in `testing`.

# Installation
After `go mod init <your_project_name>` and `go mod tidy` just run:
```
go get github.com/peliseev/testserver
```
# Usage
The common way to use this framework is as follows:

1. Create new instance of `TestServer`
2. Add some expectations
3. Start `TestServer`
4. Create an instance of the object under test
5. Call the methods under test
6. Check the results

HTTP calls are expected in the order in which you have declared them

For example:
```go
func TestFunc(t *testing.T) {
    // Create new instance
    ts := testserver.New(t)
    
    // Add some expectations
    ts.Add(EXPECT().Method("GET").Path("/api/profile").
		RespWithStatus(200).
		RespWithBody(profile))
	
    ts.Add(EXPECT().Method("POST").Path("/api/order").
		RespWithStatus(200).
		RespWithBody(done).Times(2))
    ...
    
    // Start test Server
    ts.Start()
    defer ts.Stop()
    
    // Create an instance of the object under test
    service := order.New(ts.URL)
    
    // Call the method under test
    gotRs, gotErr := service.CreateOrder(args...)
    
    // Check the results
    if wantRs != gotRs {
        t.Error()
    }
    if wantErr != gotErr {
        t.Error()
    }
}
```

You can find more examples of usage [here](example_test.go)
