package tools

import "context"

type AddInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddOutput struct {
	Result int `json:"result"`
}

func Add(_ context.Context, in *AddInput) (*AddOutput, error) {
	return &AddOutput{Result: in.A + in.B}, nil
}
