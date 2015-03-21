package rdoclient;

import (
	"encoding/json"
	"log"
	"net/http"
	"io/ioutil"
	"strings"
	"fmt"
)



func call(address string, method string, id interface{}, params interface{})(JsonResult, error){
    result := JsonResult{}

    data, err := json.Marshal(map[string]interface{}{
	"jsonrpc": "2.0",
        "method": method,
        "id":     id,
        "params": params,
    })
    if err != nil {
        log.Fatalf("Marshal: %v", err)
	return result, err
    }
    resp, err := http.Post(address,
        "application/json", strings.NewReader(string(data)))
    if err != nil {
        log.Fatalf("Post: %v", err)
	return result, err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("ReadAll: %v", err)
	return result, err
    }



    //result := make(map[string]interface{})
    err = json.Unmarshal(body, &result)
    if err != nil {
        log.Fatalf("Unmarshal: %v", err)
	return result, err
    }
    //log.Println(result)
    fmt.Println(string(string(body)))
    return result, nil
}

type GenIntParams struct {
	ApiKey string `json:"apiKey"`
	N int `json:"n"`
	Min int `json:"min"`
	Max int `json:"max"`
	Replacement bool `json:"replacement"`
}

// {"jsonrpc":"2.0","result":{"random":{"data":[6],"completionTime":"2015-03-19 18:20:51Z"},"bitsUsed":3,"bitsLeft":249761,"requestsLeft":967,"advisoryDelay":0},"id":15}
type JsonResult struct {
	Result RdoResult
	Data []int
}

type RdoResult struct {
	Random RandomData
}

type RandomData struct {
	Data []interface{} `json:"data"`
	BitsUsed int `json:"bitsUsed"`
}


func GenerateIntegers(n, min, max int, replacement bool) []int {

	intParams := GenIntParams{ApiKey: "26a65c82-7091-45f7-af12-414589392fb0", N: n, Min: min, Max: max, Replacement: replacement}
	result,err := call("https://api.random.org/json-rpc/1/invoke", "generateIntegers", 15, intParams)
	if err != nil {
		fmt.Println(err)
	}

	resultints := make([]int,1)

	for i := range result.Result.Random.Data {
		resultints = append(resultints,int(result.Result.Random.Data[i].(float64)))
	}

	return resultints


}


