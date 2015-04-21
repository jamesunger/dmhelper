package rdoclient

import (
	"testing"
	"fmt"
)


func TestGenerateIntegers(t *testing.T) {
	myints,err := GenerateIntegers(10,1,10,true)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if len(myints) != 10 {
		t.Log("Expected 10 integers but got ", len(myints))
		t.Fail()
	}

	fmt.Println("OK GenerateDecimalIntegers: ", myints)

}

func TestGenerateDecimalFractions(t *testing.T) {
	myfs,err := GenerateDecimalFractions(10,10,true)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(myfs) != 10 {
		t.Log("Expected 10 floating points but got ", len(myfs))
		t.Fail()
	}

	fmt.Println("OK GenerateDecimalFractions: ", myfs)

}

func TestGenerateGaussians(t *testing.T) {
	myfs,err := GenerateGaussians(10,100,20,5)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(myfs) != 10 {
		t.Log("Expected 10 floating points but got ", len(myfs))
		t.Fail()
	}

	fmt.Println("OK GenerateGaussians: ", myfs)

}

func TestGenerateStrings(t *testing.T) {
	mys,err := GenerateStrings(10,10,"abcdefghijklmnopqrstuvwxyz",true)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(mys) != 10 {
		t.Log("Expected 10 strings but got ", len(mys))
		t.Fail()
	}

	fmt.Println("OK GenerateStrings: ", mys)

}

func TestGenerateUUIDs(t *testing.T) {
	mys,err := GenerateUUIDs(10)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(mys) != 10 {
		t.Log("Expected 10 strings but got ", len(mys))
		t.Fail()
	}

	fmt.Println("OK GenerateUUIDs: ", mys)

}

func TestGenerateBlobs(t *testing.T) {
	mys,err := GenerateBlobs(10,8)

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}


	if len(mys) != 10 {
		t.Log("Expected 10 sets of 5 bytes but got ", len(mys))
		t.Fail()
	}

	fmt.Println("OK GenerateBlobs: ", mys)

}
