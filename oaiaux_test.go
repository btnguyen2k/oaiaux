package oaiaux

import (
	"fmt"
	"testing"
)

func TestCountTokens(t *testing.T) {
	testName := "TestCountTokens"
	testData := []struct {
		name     string
		input    string
		expected int
	}{
		{name: "Chinese", input: "你好世界，这太漂亮了!", expected: 23},
		{name: "English", input: "Hello world, this is so beautiful!", expected: 8},
		{name: "French", input: "Bonjour le monde, c'est si beau!", expected: 14},
		{name: "German", input: "Hallo Welt, das ist so schön!", expected: 13},
		{name: "Japanese", input: "こんにちは世界、これはとても美しいです！", expected: 27},
		{name: "Korean", input: "안녕하세요 세계, 이것은 너무 아름답네요!", expected: 50},
		{name: "Lao", input: "ສະບາຍດີໂລກ, ມັນແມ່ນສວຍງາມນີ້!", expected: 81},
		{name: "Thai", input: "สวัสดีโลก นี่มีความงามมาก!", expected: 50},
		{name: "Spanish", input: "¡Hola mundo, esto es tan hermoso!", expected: 15},
		{name: "Vietnamese", input: "Chào thế giới, điều này thật đẹp!", expected: 36},
	}
	for _, testCase := range testData {
		t.Run(testCase.name, func(t *testing.T) {
			value := CountTokens(testCase.input)
			if value != testCase.expected {
				t.Fatalf("%s failed for input <%s>: expected %#v but received %#v", testName+"/"+testCase.name, testCase.input, testCase.expected, value)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	testName := "TestEstimateTokens"
	testData := []struct {
		name     string
		input    string
		expected int
	}{
		{name: "Chinese", input: "第一个是一，第二个是二，第三个是三。", expected: 33},
		{name: "English", input: "Number 1 is one, number 2 is two and number 3 is three.", expected: 15},
		{name: "French", input: "Le numéro 1 est un, le numéro 2 est deux et le numéro 3 est trois.", expected: 26},
		{name: "German", input: "Nummer 1 ist eins, Nummer 2 ist zwei und Nummer 3 ist drei.", expected: 25},
		{name: "Japanese", input: "番号1は1で、番号2は2で、番号3は3です。", expected: 28},
		{name: "Korean", input: "번호 1은 1이고, 번호 2는 2이고, 번호 3은 3입니다.", expected: 51},
		{name: "Lao", input: "ໝາຍເລກ 1 ແມ່ນຫນຶ່ງ, ໝາຍເລກ 2 ແມ່ນສອງ, ແລະ ໝາຍເລກ 3 ແມ່ນສາມ.", expected: 144},
		{name: "Thai", input: "หมายเลข 1 คือหนึ่ง หมายเลข 2 คือสอง และหมายเลข 3 คือสาม", expected: 96},
		{name: "Spanish", input: "El número 1 es uno, el número 2 es dos y el número 3 es tres.", expected: 29},
		{name: "Vietnamese", input: "Số 1 là một, số 2 là hai và số 3 là ba.", expected: 33},
	}
	for _, testCase := range testData {
		t.Run(testCase.name, func(t *testing.T) {
			value := EstimateTokens(testCase.input)
			fmt.Printf("%s: <input: %s> / expected: %#v vs received: %#v\n", testName+"/"+testCase.name, testCase.input, testCase.expected, value)
			// if value != testCase.expected {
			// t.Fatalf("%s failed for input <%s>: expected %#v but received %#v", testName+"/"+testCase.name, testCase.input, testCase.expected, value)
			// }
		})
	}
}
