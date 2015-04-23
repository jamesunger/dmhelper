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

const (
	RDO_URL = "https://api.random.org/json-rpc/1/invoke"
	INT_MIN = -1000000000
	INT_MAX = 1000000000
	MAXNUM = 10000
	MAXUUIDS = 1000
	MAX_DEC_PLACES = 20
	GAUS_MAX_MEAN = 1000000
	GAUS_MIN_MEAN = -1000000
	GAUS_MIN_STDDEV = -1000000
	GAUS_MAX_STDDEV = 1000000
	GAUS_SIG_DIG = 20
	STR_MAX = 20
	STR_CHARS_MAX = 80
	BLOB_MAX = 100
	BLOB_MAX_SIZE = 1048576
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


    if result.Error.Code != 0 {
	//log.Fatalf("Got error from random.org: " + result.Error)
	return result, errors.New("Got error from random.org: " + fmt.Sprintf("%d",result.Error.Code) + ": " + result.Error.Message)
    }

    //log.Println(result)
    //fmt.Println(string(body))
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

type GenStringParams struct {
	ApiKey string `json:"apiKey"`
	N int `json:"n"`
	Length int `json:"length"`
	Characters string `json:"characters"`
	Replacement bool `json:"replacement"`
}

type GenUUIDsParams struct {
	ApiKey string `json:"apiKey"`
	N int `json:"n"`
}

type GenBlobsParams struct {
	ApiKey string `json:"apiKey"`
	N int `json:"n"`
	Size int `json:"size"`
}

type GetUsageParams struct {
	ApiKey string `json:"apiKey"`
}






// {"jsonrpc":"2.0","result":{"random":{"data":[6],"completionTime":"2015-03-19 18:20:51Z"},"bitsUsed":3,"bitsLeft":249761,"requestsLeft":967,"advisoryDelay":0},"id":15}
//"error":{"code":201,"message":"Parameter 'size' has illegal value","data":["size"]},"id":15}
type JsonResult struct {
	Result RdoResult
	Data []int
	Error JsonError `json:"Error"`


}

type RdoResult struct {
	Random RandomData

	// usage info, mostly unused
	Status string `json:"status"`
	CreationTime string `json:"creationTime"`
	BitsLeft int `json:"bitsLeft"`
	RequestsLeft int `json:"requestsLeft"`
	TotalBits int `json:"totalBits"`
	TotalRequests int `json:"totalRequests"`
}

type RandomData struct {
	Data []interface{} `json:"data"`
	BitsUsed int `json:"bitsUsed"`
}


type JsonError struct {
	Code int `json:"code"`
	Message string `json:"message"`
}


func GenerateIntegers(apikey string, n, min, max int, replacement bool) ([]int,error) {

	resultints := make([]int,0)

	if n > MAXNUM {
		return resultints,errors.New(fmt.Sprintf("Max numbers to request is %d.",MAXNUM))
	}

	if max > INT_MAX {
		return resultints,errors.New(fmt.Sprintf("Max integer is too large (%d).",INT_MAX))
	}

	if min < INT_MIN {
		return resultints,errors.New(fmt.Sprintf("Max integer is too small (%d).",INT_MIN))
	}

	intParams := GenIntParams{ApiKey: apikey, N: n, Min: min, Max: max, Replacement: replacement}
	result,err := call(RDO_URL, "generateIntegers", 15, intParams)
	if err != nil {
		//fmt.Println(err)
		return resultints,err
	}


	for i := range result.Result.Random.Data {
		resultints = append(resultints,int(result.Result.Random.Data[i].(float64)))
	}

	return resultints,nil


}

func GenerateDecimalFractions(apikey string, n, places int,replacement bool) ([]float64,error) {

	resultfs := make([]float64,0)

	if n < 1 || n > MAXNUM {
		return resultfs,errors.New(fmt.Sprintf("Max numbers to request is %d.",MAXNUM))
	}

	if places < 1 || places > MAX_DEC_PLACES {
		return resultfs,errors.New(fmt.Sprintf("Places must be between 1 and %d.",MAX_DEC_PLACES))
	}

	decFracParams := GenDecFracParams{ApiKey: apikey, N: n, DecimalPlaces: places, Replacement: replacement}
	result,err := call(RDO_URL, "generateDecimalFractions", 15, decFracParams)
	if err != nil {
		//fmt.Println(err)
		return resultfs,err
	}


	for i := range result.Result.Random.Data {
		resultfs = append(resultfs,result.Result.Random.Data[i].(float64))
	}

	return resultfs,nil


}

func GenerateGaussians(apikey string, n, mean, standardDeviation, significantDigits int) ([]float64,error) {
	resultfs := make([]float64,0)

	if n < 1 || n > MAXNUM {
		return resultfs,errors.New(fmt.Sprintf("Max numbers to request is %d.",MAXNUM))
	}

	if mean < GAUS_MIN_MEAN || mean > GAUS_MAX_MEAN {
		return resultfs,errors.New(fmt.Sprintf("Mean must be between -%d and %d",GAUS_MIN_MEAN,GAUS_MAX_MEAN))
	}

	if standardDeviation < GAUS_MIN_STDDEV || standardDeviation > GAUS_MAX_STDDEV {
		return resultfs,errors.New(fmt.Sprintf("Standard deviation must be between %d and %d",GAUS_MIN_STDDEV,GAUS_MAX_STDDEV))
	}

	if significantDigits < 1 || significantDigits > GAUS_SIG_DIG {
		return resultfs,errors.New(fmt.Sprintf("Significantdigits must be between 1 and %d.",GAUS_SIG_DIG))
	}

	genGaussianParams := GenGaussianParams{ApiKey: apikey, N: n, Mean: mean, StandardDeviation: standardDeviation, SignificantDigits: significantDigits}
	result,err := call(RDO_URL, "generateGaussians", 15, genGaussianParams)
	if err != nil {
		//fmt.Println(err)
		return resultfs,err
	}


	for i := range result.Result.Random.Data {
		resultfs = append(resultfs,result.Result.Random.Data[i].(float64))
	}

	return resultfs,nil

}

func GenerateStrings(apikey string, n, length int, characters string, replacement bool) ([]string,error) {
	results := make([]string,0)

	if n < 1 || n > MAXNUM {
		return results,errors.New(fmt.Sprintf("Max numbers to request is %d.",MAXNUM))
	}

	if length > STR_MAX {
		return results,errors.New(fmt.Sprintf("String length must 1 to %d",STR_MAX))
	}

	if len(characters) > STR_CHARS_MAX {
		return results,errors.New(fmt.Sprintf("String chars length must 1 to %d",STR_CHARS_MAX))
	}


	genStringParams := GenStringParams{ApiKey: apikey, N: n, Length: length, Characters: characters, Replacement: replacement}
	result,err := call(RDO_URL, "generateStrings", 15, genStringParams)
	if err != nil {
		//fmt.Println(err)
		return results,err
	}


	for i := range result.Result.Random.Data {
		results = append(results,result.Result.Random.Data[i].(string))
	}

	return results,nil

}

func GenerateUUIDs(apikey string, n int) ([]string,error) {
	results := make([]string,0)

	if n < 1 || n > MAXUUIDS {
		return results,errors.New(fmt.Sprintf("Max numbers to request is %d.",MAXUUIDS))
	}


	genUUIDsParams := GenUUIDsParams{ApiKey: apikey, N: n}
	result,err := call(RDO_URL, "generateUUIDs", 15, genUUIDsParams)
	if err != nil {
		//fmt.Println(err)
		return results,err
	}


	for i := range result.Result.Random.Data {
		results = append(results,result.Result.Random.Data[i].(string))
	}

	return results,nil

}

func GenerateBlobs(apikey string, n, size int) ([]string,error) {
	results := make([]string,0)

	if n < 1 || n > BLOB_MAX {
		return results,errors.New(fmt.Sprintf("Max numbers to request is %d.",BLOB_MAX))
	}


	genBlobsParams := GenBlobsParams{ApiKey: apikey, N: n, Size: size}
	result,err := call(RDO_URL, "generateBlobs", 15, genBlobsParams)
	if err != nil {
		//fmt.Println(err)
		return results,err
	}


	for i := range result.Result.Random.Data {
		results = append(results,result.Result.Random.Data[i].(string))
	}

	return results,nil

}

func GetUsage(apikey string) (JsonResult,error) {

	getUsageParams := GetUsageParams{ApiKey: apikey}
	usageresult,err := call(RDO_URL, "getUsage", 15, getUsageParams)
	if err != nil {
		//fmt.Println(err)
		return usageresult,err
	}

	return usageresult,nil

}



