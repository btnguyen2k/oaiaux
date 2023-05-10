// package aux provides helper utilities to work with OpenAI API.
package oaiaux

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/btnguyen2k/consu/gjrc"
	"github.com/btnguyen2k/consu/reddo"
	"github.com/tiktoken-go/tokenizer"
)

// Flavor specifies which OpenAI "flavor" to use (currently available: platform.openai.com and Azure OpenAI).
type Flavor int

const (
	PlatformOpenAI Flavor = iota
	AzureOpenAI
)

var (
	ErrOptionNotFound = errors.New("option not found")
)

// Option contains an option/parameter to supply to API/function calls.
type Option struct {
	Key   string
	Value interface{}
}

// AsString returns the option value as string.
func (o Option) AsString() (string, error) {
	return reddo.ToString(o.Value)
}

// OptionList combines individual Option instances for convenient use.
type OptionList []Option

// GetString finds an option matching 'key' and return its value as string.
func (ol OptionList) GetString(key string) (string, error) {
	for _, o := range ol {
		if o.Key == key {
			return o.AsString()
		}
	}
	return "", ErrOptionNotFound
}

/*----------------------------------------------------------------------*/

// NewClient creates a new Client instance.
//
// 'flavor' parameter specifies the Flavor of the Client to be created.
func NewClient(flavor Flavor, opts ...Option) (Client, error) {
	switch flavor {
	case AzureOpenAI:
		baseClient := &BaseClient{
			gjrc: gjrc.NewGjrc(nil, 60*time.Second),
			opts: opts,
		}
		client := &AzureOpenAIClient{BaseClient: baseClient}
		return client, client.Init()
	case PlatformOpenAI:
		baseClient := &BaseClient{
			gjrc: gjrc.NewGjrc(nil, 60*time.Second),
			opts: opts,
		}
		client := &PlatformOpenAIClient{BaseClient: baseClient}
		return client, client.Init()
	}
	return nil, fmt.Errorf("unknown flavor %#v", flavor)
}

type PromptInput struct {
	Model            string         `json:"model,omitempty"`
	Prompt           string         `json:"prompt"`
	MaxTokens        int            `json:"max_tokens"`
	Temperature      float64        `json:"temperature"`
	TopP             float64        `json:"top_p"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             string         `json:"user,omitempty"`
	N                int            `json:"n"`
	Stream           bool           `json:"stream"`
	LogProbs         int            `json:"logprobs"`
	Suffix           string         `json:"suffix,omitempty"`
	Echo             bool           `json:"echo"`
	Stop             []string       `json:"stop,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty"`
	FrequencyPenalty float64        `json:"frequency_penalty"`
	BestOf           int            `json:"best_of"`
}

type CompletionsOutput struct {
	Error      error  `json:"-"`
	StatusCode int    `json:"-"`
	Id         string `json:"id"`
	Object     string `json:"object"`
	Created    int64  `json:"created"`
	Model      string `json:"model"`
	Usage      *struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Text         string                 `json:"text"`
		Index        int                    `json:"index"`
		FinishReason string                 `json:"finish_reason"`
		LogProbs     map[string]interface{} `json:"logprobs"`
	} `json:"choices"`
}

// Client captures OpenAI REST API.
type Client interface {
	// Completions make a 'completions' API call and returns the completions output.
	Completions(prompt *PromptInput) *CompletionsOutput
}

const (
	OptAzureResourceName = "azure-resource-name"
	OptAzureApiVersion   = "azure-api-version"
	OptAzureApiKey       = "azure-api-key"

	OptOpenAIApiKey       = "openai-api-key"
	OptOpenAIOrganization = "openai-Organization"
)

type BaseClient struct {
	gjrc *gjrc.Gjrc
	opts OptionList
}

func (bc *BaseClient) preparePrompt(prompt *PromptInput) *PromptInput {
	if prompt.MaxTokens <= 0 {
		prompt.MaxTokens = 100
	}
	if prompt.N < 1 {
		prompt.N = 1
	}
	if prompt.BestOf < prompt.N {
		prompt.BestOf = prompt.N
	}
	return prompt
}

func (bc *BaseClient) buildCompletions(resp *gjrc.GjrcResponse) *CompletionsOutput {
	completions := &CompletionsOutput{Error: resp.Error()}
	if completions.Error == nil {
		err := resp.Unmarshal(completions)
		completions.Error = err
	}
	completions.StatusCode = resp.StatusCode()
	return completions
}

// AzureOpenAIClient is AzureOpenAI-flavor of Client.
type AzureOpenAIClient struct {
	*BaseClient
	resourceName, apiVersion, apiKey string
}

// Init should be called to initialize the client before any API call.
func (c *AzureOpenAIClient) Init() error {
	var err error

	c.resourceName, err = c.opts.GetString(OptAzureResourceName)
	if err != nil || c.resourceName == "" {
		return fmt.Errorf("cannot parse setting <%s> %s", OptAzureResourceName, err)
	}

	c.apiKey, err = c.opts.GetString(OptAzureApiKey)
	if err != nil || c.apiKey == "" {
		return fmt.Errorf("cannot parse setting <%s> %s", OptAzureApiKey, err)
	}

	c.apiVersion, err = c.opts.GetString(OptAzureApiVersion)
	if err != nil || c.apiVersion == "" {
		c.apiVersion = "2023-03-15-preview"
	}

	return nil
}

func (c *AzureOpenAIClient) buildUrlCompletions(prompt *PromptInput) string {
	url := "https://{azure-resource-name}.openai.azure.com/openai/deployments/{model}/completions?api-version={azure-api-version}"
	url = strings.ReplaceAll(url, "{azure-resource-name}", c.resourceName)
	url = strings.ReplaceAll(url, "{model}", prompt.Model)
	url = strings.ReplaceAll(url, "{azure-api-version}", c.apiVersion)
	return url
}

// Completions implements Client.Completions
func (c *AzureOpenAIClient) Completions(prompt *PromptInput) *CompletionsOutput {
	apiUrl := c.buildUrlCompletions(prompt)
	header := http.Header{}
	header.Set("api-key", c.apiKey)
	prompt = c.preparePrompt(prompt)
	resp := c.gjrc.PostJson(apiUrl, prompt, gjrc.RequestMeta{Header: header})
	return c.buildCompletions(resp)
}

// PlatformOpenAIClient is platform.openai.com-flavor of Client.
type PlatformOpenAIClient struct {
	*BaseClient
	apiKey, organization string
}

func (c *PlatformOpenAIClient) Init() error {
	var err error

	c.apiKey, err = c.opts.GetString(OptOpenAIApiKey)
	if err != nil || c.apiKey == "" {
		return fmt.Errorf("cannot parse setting <%s> %s", OptAzureApiKey, err)
	}

	c.organization, err = c.opts.GetString(OptOpenAIOrganization)

	return nil
}

func (c *PlatformOpenAIClient) buildUrlCompletions(prompt *PromptInput) string {
	url := "https://api.openai.com/v1/completions"
	return url
}

// Completions implements Client.Completions
func (c *PlatformOpenAIClient) Completions(prompt *PromptInput) *CompletionsOutput {
	apiUrl := c.buildUrlCompletions(prompt)
	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.apiKey)
	if c.organization != "" {
		header.Set("OpenAI-Organization", c.organization)
	}
	prompt = c.preparePrompt(prompt)
	resp := c.gjrc.PostJson(apiUrl, prompt, gjrc.RequestMeta{Header: header})
	return c.buildCompletions(resp)
}

/*----------------------------------------------------------------------*/

// CountTokens returnes the number of BPE tokens for an input string. If error, -1 is returned.
func CountTokens(input string, opts ...Option) int {
	var optList OptionList = opts
	var enc tokenizer.Codec

	model, err := optList.GetString("model")
	if model != "" && err == nil {
		enc, err = tokenizer.ForModel(tokenizer.Model(model))
	}
	if enc == nil {
		encoding, err := optList.GetString("encoding")
		if encoding != "" && err == nil {
			enc, err = tokenizer.Get(tokenizer.Encoding(encoding))
		}
	}
	if enc == nil {
		enc, err = tokenizer.Get(tokenizer.P50kBase)
		if enc == nil || err != nil {
			return -1
		}
	}

	ids, _, _ := enc.Encode(input)
	return len(ids)
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
			numBytes := len([]byte(nw))
			numNonWords += numBytes
		}
	}

	numBytes := len([]byte(input))
	return ((numWords*4/3 + numNonWords) + numBytes/4) / 2
}
