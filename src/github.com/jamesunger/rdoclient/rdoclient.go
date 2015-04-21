package rdoclient

import (
	"encoding/json"
	"log"
	"net/http"
	"io/ioutil"
	"strings"
	"fmt"
	"errors"
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
    //fmt.Println(string(string(body)))
    return result, nil
}

type GenIntParams struct {
	ApiKey string `json:"apiKey"`
	N int `json:"n"`
	Min int `json:"min"`
	Max int `json:"max"`
	Replacement bool `json:"replacement"`
}

type GenDecFracParams struct {
	ApiKey string `json:"apiKey"`
	N int `json:"n"`
	DecimalPlaces int `json:"decimalPlaces"`
	Replacement bool `json:"replacement"`
}

type GenGaussianParams struct {
	ApiKey string `json:"apiKey"`
	N int `json:"n"`
	Mean int `json:"mean"`
	StandardDeviation int `json:"standardDeviation"`
	SignificantDigits int `json:"significantDigits"`
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


func GenerateIntegers(n, min, max int, replacement bool) ([]int,error) {

	resultints := make([]int,0)

	if n > 10000 {
		return resultints,errors.New("Max numbers to request is 1000.")
	}

	if max > 1000000000 {
		return resultints,errors.New("Max integer is too large (1000000000).")
	}

	if min < -1000000000 {
		return resultints,errors.New("Max integer is too small (-1000000000).")
	}

	intParams := GenIntParams{ApiKey: "26a65c82-7091-45f7-af12-414589392fb0", N: n, Min: min, Max: max, Replacement: replacement}
	result,err := call("https://api.random.org/json-rpc/1/invoke", "generateIntegers", 15, intParams)
	if err != nil {
		fmt.Println(err)
	}


	for i := range result.Result.Random.Data {
		resultints = append(resultints,int(result.Result.Random.Data[i].(float64)))
	}

	return resultints,nil


}

func GenerateDecimalFractions(n, places int,replacement bool) ([]float64,error) {

	resultfs := make([]float64,0)

	if n < 1 || n >10000 {
		return resultfs,errors.New("Request number must be between 1 and 10000")
	}

	if places < 1 || places > 20 {
		return resultfs,errors.New("Places must be between 1 and 20.")
	}

	decFracParams := GenDecFracParams{ApiKey: "26a65c82-7091-45f7-af12-414589392fb0", N: n, DecimalPlaces: places, Replacement: replacement}
	result,err := call("https://api.random.org/json-rpc/1/invoke", "generateDecimalFractions", 15, decFracParams)
	if err != nil {
		fmt.Println(err)
	}


	for i := range result.Result.Random.Data {
		resultfs = append(resultfs,result.Result.Random.Data[i].(float64))
	}

	return resultfs,nil


}

func GenerateGaussians(n, mean, standardDeviation, significantDigits int) ([]float64,error) {
	resultfs := make([]float64,0)

	if n < 1 || n > 10000 {
		return resultfs,errors.New("Number of guassians to request must be between 1 and 10000")
	}

	if mean < -1000000 || mean > 1000000 {
		return resultfs,errors.New("Mean must be between -1000000 and 1000000")
	}

	if standardDeviation < -1000000 || standardDeviation > 1000000 {
		return resultfs,errors.New("Standard deviation must be between -1000000 and 1000000")
	}

	if significantDigits < 1 || significantDigits > 20 {
		return resultfs,errors.New("Significantdigits must be between 1 and 20.")
	}

	genGaussianParams := GenGaussianParams{ApiKey: "26a65c82-7091-45f7-af12-414589392fb0", N: n, Mean: mean, StandardDeviation: standardDeviation, SignificantDigits: significantDigits}
	result,err := call("https://api.random.org/json-rpc/1/invoke", "generateGaussians", 15, genGaussianParams)
	if err != nil {
		fmt.Println(err)
	}


	for i := range result.Result.Random.Data {
		resultfs = append(resultfs,result.Result.Random.Data[i].(float64))
	}

	return resultfs,nil


}




