/*
Package aux provides helper utilities to work with OpenAI API.

Sample usage:

	package main

	import (
		"fmt"
		"os"

		"github.com/btnguyen2k/oaiaux"
	)

	func main() {
		// Azure OpenAI client requires 2 mandatory settings: Azure resource name and Azure OpenAI API key
		clientAOAI, err := oaiaux.NewClient(oaiaux.AzureOpenAI,
			oaiaux.Option{Key: oaiaux.OptAzureResourceName, Value: os.Getenv("AZURE_OPENAI_RESOURCE_NAME")},
			oaiaux.Option{Key: oaiaux.OptAzureApiKey, Value: os.Getenv("AZURE_OPENAI_API_KEY")},
		)
		if err != nil {
			panic(err)
		}

		// Platform.OpenAI.Com client requires 1 mandatory setting: OpenAI API key
		// and one optional setting: OpenAI organization id
		clientOpenAI, err := oaiaux.NewClient(oaiaux.PlatformOpenAI,
			oaiaux.Option{Key: oaiaux.OptOpenAIApiKey, Value: os.Getenv("OPENAI_API_KEY")},
			oaiaux.Option{Key: oaiaux.OptOpenAIOrganization, Value: os.Getenv("OPENAI_ORGANIZATION_ID")},
		)
		if err != nil {
			panic(err)
		}

		// build prompt
		// note: for Azure OpenAI service, supply the model deployment name as the value of the "Model" parameter
		prompt := &oaiaux.PromptInput{
			Model:     "text-davinci-003",
			Prompt:    "Write a tagline for an ice cream shop.",
			MaxTokens: 250,
		}
		// get completions
		completions := clientAOAI.Completions(prompt)
		if completions.Error != nil {
			panic(fmt.Errorf("Error: %s\n", completions.Error))
		} else if completions.StatusCode != 200 {
			panic(fmt.Errorf("Error: %#v\n", completions.StatusCode))
		} else {
			for i, c := range completions.Choices {
				fmt.Printf("Completion<%#v/%#v>: %#v\n", i, c.FinishReason, c.Text)
			}
		}

		// prepare the input for embeddings API call
		embeddingsInput := &oaiaux.EmbeddingsInput{
			Model: "text-embedding-ada-002",
			Input: "Cool down with our delicious treats!",
		}
		// call API to calculate embeddings vector
		embeddings := clientOpenAI.Embeddings(input)
		if embeddings.Error != nil {
			panic(fmt.Errorf("Error: %s\n", embeddings.Error))
		} else if embeddings.StatusCode != 200 {
			panic(fmt.Errorf("Error: %#v\n", embeddings.StatusCode))
		} else {
			for i, d := range embeddings.Data {
				fmt.Printf("Embeddings<%#v/%#v>: length %#v\n", i, d.Object, len(d.Embedding))
			}
		}


		// prepare the input for chat-completions API call
		chatPrompt := &oaiaux.ChatPromptInput{
			Model:       "gpt-3.5-turbo",
			Temperature: 0.7,
			Messages: []oaiaux.ChatMessage{
				{Role: "system", Content: "You are a friendly assistant."},
				{Role: "user", Content: "What is GPT?"},
			},
			MaxTokens: 150,
		}
		// get completions
		chatCompletions := clientOpenAI.ChatCompletions(chatPrompt)
		if chatCompletions.Error != nil {
			panic(fmt.Errors("Error: %s\n", chatCompletions.Error))
		} else if completions.StatusCode != 200 {
			panic(fmt.Errors("Error: %#v\n", chatCompletions.StatusCode))
		} else {
			for i, c := range chatCompletions.Choices {
				fmt.Printf("Completion<%#v/%#v>: %#v\n", i, c.FinishReason, c.Message)
			}
		}

		// note: for Azure OpenAI service, supply the model deployment name as the value of the "Model" parameter
		chatPrompt.Model = "gpt-35-turbo" // Azure OpenAI currently does not allow character '.' in the model deployment name
		chatCompletions = clientAOAI.ChatCompletions(chatPrompt)
		if chatCompletions.Error != nil {
			panic(fmt.Errors("Error: %s\n", chatCompletions.Error))
		} else if completions.StatusCode != 200 {
			panic(fmt.Errors("Error: %#v\n", chatCompletions.StatusCode))
		} else {
			for i, c := range chatCompletions.Choices {
				fmt.Printf("Completion<%#v/%#v>: %#v\n", i, c.FinishReason, c.Message)
			}
		}
	}
*/
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

const (
	// Version of oaiaux
	Version = "0.1.2"
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

type BaseResponse struct {
	Error      error `json:"-"`
	StatusCode int   `json:"-"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type ChatPromptInput struct {
	Model            string         `json:"model,omitempty"`
	Messages         []ChatMessage  `json:"messages"`
	Temperature      float64        `json:"temperature"`
	TopP             float64        `json:"top_p"`
	N                int            `json:"n"`
	Stream           bool           `json:"stream"`
	Stop             []string       `json:"stop,omitempty"`
	MaxTokens        int            `json:"max_tokens"`
	PresencePenalty  float64        `json:"presence_penalty"`
	FrequencyPenalty float64        `json:"frequency_penalty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             string         `json:"user,omitempty"`
}

type ChatCompletionsOutput struct {
	BaseResponse `json:"-"`
	Id           string `json:"id"`
	Object       string `json:"object"`
	Created      int64  `json:"created"`
	Model        string `json:"model"`
	Usage        *struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message      ChatMessage `json:"message"`
		Index        int         `json:"index"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
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
	BaseResponse `json:"-"`
	Id           string `json:"id"`
	Object       string `json:"object"`
	Created      int64  `json:"created"`
	Model        string `json:"model"`
	Usage        *struct {
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

type EmbeddingsInput struct {
	Model     string `json:"model,omitempty"`
	Input     string `json:"input"`
	InputType string `json:"input_type,omitempty"`
	User      string `json:"user,omitempty"`
}

type EmbeddingsOutput struct {
	BaseResponse `json:"-"`
	Object       string `json:"object"`
	Model        string `json:"model"`
	Data         []struct {
		Index     int    `json:"index"`
		Object    string `json:"object"`
		Embedding Vector `json:"embedding"`
	} `json:"data"`
	Usage *struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Client captures OpenAI REST API.
type Client interface {
	// Completions make a 'completions' API call and returns the completions output.
	Completions(prompt *PromptInput) *CompletionsOutput

	// ChatCompletions make a 'chat-completions' API call and returns the completions output.
	ChatCompletions(prompt *ChatPromptInput) *ChatCompletionsOutput

	// Embeddings make an 'embeddings' API call and returns the embeddings output.
	Embeddings(input *EmbeddingsInput) *EmbeddingsOutput
}

const (
	// OptAzureResourceName specifies the Azure OpenAI's resource-name.
	OptAzureResourceName = "azure-resource-name"
	// OptAzureApiVersion specifies the version of Azure OpenAI to use (default "2023-03-15-preview").
	OptAzureApiVersion = "azure-api-version"
	// OptAzureApiKey specifies the API key used to call Azure OpenAI APIs.
	OptAzureApiKey = "azure-api-key"

	// OptOpenAIApiKey specifies the API key used to call OpenAI APIs.
	OptOpenAIApiKey = "openai-api-key"
	// OptOpenAIApiKey specifies the OpenAI's organization name.
	OptOpenAIOrganization = "openai-organization"
	// OptOpenAIBaseUrl specifies the custom base url for OpenAI APIs (for example "http://localhost:5123").
	OptOpenAIBaseUrl = "openai-base-url"
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

	if 0.0 == prompt.Temperature && 0.0 == prompt.TopP {
		prompt.Temperature = 1.0
		prompt.TopP = 1.0
	}
	if prompt.Temperature < 0.0 || prompt.Temperature > 1.0 {
		prompt.Temperature = 1.0
	}
	if prompt.TopP < 0.0 || prompt.TopP > 1.0 {
		prompt.TopP = 1.0
	}
	if 0.0 < prompt.Temperature && prompt.Temperature < 1.0 {
		prompt.TopP = 1.0
	}
	if 0.0 < prompt.TopP && prompt.TopP < 1.0 {
		prompt.Temperature = 1.0
	}

	return prompt
}

func (bc *BaseClient) prepareChatPrompt(prompt *ChatPromptInput) *ChatPromptInput {
	if prompt.MaxTokens <= 0 {
		prompt.MaxTokens = 100
	}
	if prompt.N < 1 {
		prompt.N = 1
	}

	if 0.0 == prompt.Temperature && 0.0 == prompt.TopP {
		prompt.Temperature = 1.0
		prompt.TopP = 1.0
	}
	if prompt.Temperature < 0.0 || prompt.Temperature > 1.0 {
		prompt.Temperature = 1.0
	}
	if prompt.TopP < 0.0 || prompt.TopP > 1.0 {
		prompt.TopP = 1.0
	}
	if 0.0 < prompt.Temperature && prompt.Temperature < 1.0 {
		prompt.TopP = 1.0
	}
	if 0.0 < prompt.TopP && prompt.TopP < 1.0 {
		prompt.Temperature = 1.0
	}

	return prompt
}

func (bc *BaseClient) buildCompletionsOutput(resp *gjrc.GjrcResponse) *CompletionsOutput {
	completions := &CompletionsOutput{BaseResponse: BaseResponse{Error: resp.Error()}}
	if completions.Error == nil {
		err := resp.Unmarshal(completions)
		completions.Error = err
	}
	completions.StatusCode = resp.StatusCode()
	return completions
}

func (bc *BaseClient) buildChatCompletionsOutput(resp *gjrc.GjrcResponse) *ChatCompletionsOutput {
	completions := &ChatCompletionsOutput{BaseResponse: BaseResponse{Error: resp.Error()}}
	if completions.Error == nil {
		err := resp.Unmarshal(completions)
		completions.Error = err
	}
	completions.StatusCode = resp.StatusCode()
	return completions
}

func (bc *BaseClient) buildEmbeddingsOutput(resp *gjrc.GjrcResponse) *EmbeddingsOutput {
	embeddings := &EmbeddingsOutput{BaseResponse: BaseResponse{Error: resp.Error()}}
	if embeddings.Error == nil {
		err := resp.Unmarshal(embeddings)
		embeddings.Error = err
	}
	embeddings.StatusCode = resp.StatusCode()
	return embeddings
}

/*----------------------------------------------------------------------*/

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

func (c *AzureOpenAIClient) buildRequestHeaders() http.Header {
	header := http.Header{}
	header.Set("api-key", c.apiKey)
	return header
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
	header := c.buildRequestHeaders()
	prompt = c.preparePrompt(prompt)
	resp := c.gjrc.PostJson(apiUrl, prompt, gjrc.RequestMeta{Header: header})
	return c.buildCompletionsOutput(resp)
}

func (c *AzureOpenAIClient) buildUrlChatCompletions(prompt *ChatPromptInput) string {
	url := "https://{azure-resource-name}.openai.azure.com/openai/deployments/{model}/chat/completions?api-version={azure-api-version}"
	url = strings.ReplaceAll(url, "{azure-resource-name}", c.resourceName)
	url = strings.ReplaceAll(url, "{model}", prompt.Model)
	url = strings.ReplaceAll(url, "{azure-api-version}", c.apiVersion)
	return url
}

// ChatCompletions implements Client.ChatCompletions
func (c *AzureOpenAIClient) ChatCompletions(prompt *ChatPromptInput) *ChatCompletionsOutput {
	apiUrl := c.buildUrlChatCompletions(prompt)
	header := c.buildRequestHeaders()
	prompt = c.prepareChatPrompt(prompt)
	resp := c.gjrc.PostJson(apiUrl, prompt, gjrc.RequestMeta{Header: header})
	return c.buildChatCompletionsOutput(resp)
}

func (c *AzureOpenAIClient) buildUrlEmbeddings(input *EmbeddingsInput) string {
	url := "https://{azure-resource-name}.openai.azure.com/openai/deployments/{model}/embeddings?api-version={azure-api-version}"
	url = strings.ReplaceAll(url, "{azure-resource-name}", c.resourceName)
	url = strings.ReplaceAll(url, "{model}", input.Model)
	url = strings.ReplaceAll(url, "{azure-api-version}", c.apiVersion)
	return url
}

// Embeddings implements Client.Embeddings
func (c *AzureOpenAIClient) Embeddings(input *EmbeddingsInput) *EmbeddingsOutput {
	apiUrl := c.buildUrlEmbeddings(input)
	header := c.buildRequestHeaders()
	resp := c.gjrc.PostJson(apiUrl, input, gjrc.RequestMeta{Header: header})
	return c.buildEmbeddingsOutput(resp)
}

/*----------------------------------------------------------------------*/

// PlatformOpenAIClient is platform.openai.com-flavor of Client.
type PlatformOpenAIClient struct {
	*BaseClient
	apiKey, organization string
	baseUrl              string
}

func (c *PlatformOpenAIClient) Init() error {
	var err error

	c.apiKey, err = c.opts.GetString(OptOpenAIApiKey)
	if err != nil || c.apiKey == "" {
		return fmt.Errorf("cannot parse setting <%s> %s", OptOpenAIApiKey, err)
	}

	c.organization, _ = c.opts.GetString(OptOpenAIOrganization)
	c.baseUrl, _ = c.opts.GetString(OptOpenAIBaseUrl)
	c.baseUrl = strings.TrimSuffix(c.baseUrl, "/")
	if c.baseUrl == "" {
		c.baseUrl = "https://api.openai.com/v1"
	}

	return nil
}

func (c *PlatformOpenAIClient) buildRequestHeaders() http.Header {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.apiKey)
	if c.organization != "" {
		header.Set("OpenAI-Organization", c.organization)
	}
	return header
}

func (c *PlatformOpenAIClient) buildUrlCompletions(prompt *PromptInput) string {
	url := c.baseUrl + "/completions"
	return url
}

// Completions implements Client.Completions
func (c *PlatformOpenAIClient) Completions(prompt *PromptInput) *CompletionsOutput {
	apiUrl := c.buildUrlCompletions(prompt)
	header := c.buildRequestHeaders()
	prompt = c.preparePrompt(prompt)
	resp := c.gjrc.PostJson(apiUrl, prompt, gjrc.RequestMeta{Header: header})
	return c.buildCompletionsOutput(resp)
}

func (c *PlatformOpenAIClient) buildUrlChatCompletions(prompt *ChatPromptInput) string {
	url := c.baseUrl + "/chat/completions"
	return url
}

// ChatCompletions implements Client.ChatCompletions
func (c *PlatformOpenAIClient) ChatCompletions(prompt *ChatPromptInput) *ChatCompletionsOutput {
	apiUrl := c.buildUrlChatCompletions(prompt)
	header := c.buildRequestHeaders()
	prompt = c.prepareChatPrompt(prompt)
	resp := c.gjrc.PostJson(apiUrl, prompt, gjrc.RequestMeta{Header: header})
	return c.buildChatCompletionsOutput(resp)
}

func (c *PlatformOpenAIClient) buildUrlEmbeddings(input *EmbeddingsInput) string {
	url := c.baseUrl + "/embeddings"
	return url
}

// Embeddings implements Client.Embeddings
func (c *PlatformOpenAIClient) Embeddings(input *EmbeddingsInput) *EmbeddingsOutput {
	apiUrl := c.buildUrlEmbeddings(input)
	header := c.buildRequestHeaders()
	resp := c.gjrc.PostJson(apiUrl, input, gjrc.RequestMeta{Header: header})
	return c.buildEmbeddingsOutput(resp)
}

/*----------------------------------------------------------------------*/

// CountTokens returnes the number of BPE tokens for an input string. If error, -1 is returned.
func CountTokens(input string, opts ...Option) int {
	var optList OptionList = opts
	var enc tokenizer.Codec

	if model, err := optList.GetString("model"); model != "" && err == nil {
		enc, err = tokenizer.ForModel(tokenizer.Model(model))
	}
	if enc == nil {
		if encoding, err := optList.GetString("encoding"); encoding != "" && err == nil {
			enc, err = tokenizer.Get(tokenizer.Encoding(encoding))
		}
	}
	if enc == nil {
		enc, _ = tokenizer.Get(tokenizer.P50kBase)
		if enc == nil {
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
