// package aux provides helper utilities to work with OpenAI API.
package oaiaux

import (
	"fmt"
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/btnguyen2k/consu/gjrc"
	tokenizer "github.com/samber/go-gpt-3-encoder"
)

// Flavor specifies which OpenAI "flavor" to use (currently available: platform.openai.com and Azure OpenAI).
type Flavor int

const (
	PlatformOpenAI Flavor = iota
	AzureOpenAI
)

// Option contains an option/parameter to supply to API calls.
type Option struct {
	Key   string
	Value interface{}
}

func NewClient(flavor Flavor, opts ...Option) (Client, error) {
	switch flavor {
	case AzureOpenAI:
		baseClient := &baseClient{
			jrc: gjrc.NewGjrc(nil, 60*time.Second),
		}
		return &AzureOpenAIClient{baseClient}, nil
	}
	return nil, fmt.Errorf("unknown flavor %#v", flavor)
}

// Client captures OpenAI REST API.
type Client interface {
}

type baseClient struct {
	jrc *gjrc.Gjrc
}

type AzureOpenAIClient struct {
	*baseClient
}

/*----------------------------------------------------------------------*/

var (
	encoder, _  = tokenizer.NewEncoder()
	encoderLock sync.Mutex
)

// CountTokens returnes the number of BPE tokens for an input string. If error, -1 is returned.
func CountTokens(input string) int {
	encoderLock.Lock()
	defer encoderLock.Unlock()
	encoded, err := encoder.Encode(input)
	if err != nil {
		return -1
	}
	return len(encoded)
}

// EstimateTokens estimates the number of tokes for an input string.
//
// This function is for testing purpose only! Use CountTokens instead.
func EstimateTokens(input string) int {
	const re1 = `[^\w\d]+`
	const re2 = `[\w\d]+`
	reWords := regexp.MustCompile(re1)
	words := reWords.Split(input, -1)
	numWords := 0
	for _, w := range words {
		if w != "" {
			numBytes := len([]byte(w))
			numWords += int(math.Ceil(float64(numBytes) / 4.0))
		}
	}

	reNonWords := regexp.MustCompile(re2)
	nonWords := reNonWords.Split(input, -1)
	numNonWords := 0
	for _, nw := range nonWords {
		if nw != "" {
			// numNonWords++
			// barr := []byte(nw)
			// for i, n := 1, len(barr); i < n; i++ {
			// 	if barr[i] != barr[i-1] {
			// 		numNonWords++
			// 	}
			// }
			numBytes := len([]byte(nw))
			numNonWords += numBytes
		}
	}

	numBytes := len([]byte(input))
	return ((numWords*4/3 + numNonWords) + numBytes/4) / 2
}
