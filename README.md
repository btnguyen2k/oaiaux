# oaiaux

[![Go Report Card](https://goreportcard.com/badge/github.com/btnguyen2k/oaiaux)](https://goreportcard.com/report/github.com/btnguyen2k/oaiaux)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/btnguyen2k/oaiaux)](https://pkg.go.dev/github.com/btnguyen2k/oaiaux)
[![Actions Status](https://github.com/btnguyen2k/oaiaux/workflows/oaiaux/badge.svg)](https://github.com/btnguyen2k/oaiaux/actions)
[![codecov](https://codecov.io/gh/btnguyen2k/oaiaux/branch/master/graph/badge.svg?token=0L23UTJHOZ)](https://codecov.io/gh/btnguyen2k/oaiaux)
[![Release](https://img.shields.io/github/release/btnguyen2k/oaiaux.svg?style=flat-square)](RELEASE-NOTES.md)

OpenAI helper for Go.

## Installation

```shell
$ go get -u github.com/btnguyen2k/oaiaux
```

## Sample usage

Import packages and initialize client instance:

```go
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

    ...
}
```

Prompt-Completions:
```go
func main() {
    ...

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

    ...
}
```

Embeddings:
```go
func main() {
    ...

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

    ...
}
```

Chat-completions:
```go
func main() {
    ...

    // build prompt
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

    ...
}
```

## Release notes

See [RELEASE-NOTES.md](RELEASE-NOTES.md).

## License

MIT - see [LICENSE.md](LICENSE.md).
