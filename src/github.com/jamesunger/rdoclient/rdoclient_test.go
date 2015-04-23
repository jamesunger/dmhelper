package rdoclient

import (
	"testing"
	"fmt"
)

const (
	apikey = "26a65c82-7091-45f7-af12-414589392fb0"
)


func TestGenerateIntegers(t *testing.T) {
	myints,err := GenerateIntegers(apikey,10,1,10,true)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if len(myints) != 10 {
		t.Log("Expected 10 integers but got ", len(myints))
		t.Fail()
	}


	if testing.Verbose() {
		fmt.Println("OK GenerateDecimalIntegers: ", myints)
	}

}

func TestGenerateDecimalFractions(t *testing.T) {
	myfs,err := GenerateDecimalFractions(apikey,10,10,true)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(myfs) != 10 {
		t.Log("Expected 10 floating points but got ", len(myfs))
		t.Fail()
	}

	if testing.Verbose() {
		fmt.Println("OK GenerateDecimalFractions: ", myfs)
	}

}

func TestGenerateGaussians(t *testing.T) {
	myfs,err := GenerateGaussians(apikey,10,100,20,5)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(myfs) != 10 {
		t.Log("Expected 10 floating points but got ", len(myfs))
		t.Fail()
	}

	if testing.Verbose() {
		fmt.Println("OK GenerateGaussians: ", myfs)
	}

}

func TestGenerateStrings(t *testing.T) {
	mys,err := GenerateStrings(apikey,10,10,"abcdefghijklmnopqrstuvwxyz",true)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(mys) != 10 {
		t.Log("Expected 10 strings but got ", len(mys))
		t.Fail()
	}

	if testing.Verbose() {
		fmt.Println("OK GenerateStrings: ", mys)
	}

}

func TestGenerateUUIDs(t *testing.T) {
	mys,err := GenerateUUIDs(apikey,10)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(mys) != 10 {
		t.Log("Expected 10 strings but got ", len(mys))
		t.Fail()
	}

	if testing.Verbose() {
		fmt.Println("OK GenerateUUIDs: ", mys)
	}

}

func TestGenerateBlobs(t *testing.T) {
	mys,err := GenerateBlobs(apikey,10,8)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(mys) != 10 {
		t.Log("Expected 10 sets of 5 bytes but got ", len(mys))
		t.Fail()
	}

	if testing.Verbose() {
		fmt.Println("OK GenerateBlobs: ", mys)
	}
}


func TestGetUsage(t *testing.T) {
	res,err := GetUsage(apikey)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if testing.Verbose() {
		fmt.Printf("Usage: status: %s, creationTime: %s, bitsLeft: %d, requestsLeft: %d, totalBits: %d, totalRequests: %d\n",res.Result.Status, res.Result.CreationTime, res.Result.BitsLeft, res.Result.RequestsLeft, res.Result.TotalBits, res.Result.TotalRequests)
	}

}

func TestInvalidkey(t *testing.T) {
	_,err := GetUsage("foobar")

	if err != nil {
		if testing.Verbose() {
			fmt.Println("OK: ", err)
		}
		//t.Fail()
	} else {
		t.Fail()
	}




}




