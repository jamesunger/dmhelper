package rdoclient

import (
	"testing"
	"fmt"
)


func TestGenerateIntegers(t *testing.T) {
	myints,err := GenerateIntegers(10,1,10,true)

	if err != nil {
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
		t.Fail()
	}


	if len(myfs) != 10 {
		t.Log("Expected 10 floating points but got ", len(myfs))
		t.Fail()
	}

	fmt.Println("OK GenerateGaussians: ", myfs)

}
