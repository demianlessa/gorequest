# GoRequest

## Disclaimer
Not all functionality from the original Node.js request module has been yet ported. WIP

## Description
GoRequest makes HTTP requests simple and idiomatic. It was implemented and designed to be simple when making HTTP calls.

## Installation
go get github.com/mscheker/gorequest

## Simple to use
```go
package main

import (
	"fmt"
	
	request "github.com/mscheker/gorequest"
)

func main() {
	resp, body, err := request.NewRequest("https://www.google.com")
	
	fmt.Printf("Response: %v \n\r", resp)
	fmt.Printf("Body: %s \n\r", string(body))
	fmt.Printf("Error: %v \n\r", err)
}
```

## Table of Contents

## Credits
