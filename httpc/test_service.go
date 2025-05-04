package httpc

import (
	"fmt"
	"reflect"
)

// User for testing
type User struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

// MultiInput for testing
type MultiInput struct {
	Value string `json:"value"`
}

// MultiOutput for testing
type MultiOutput struct {
	Result string `json:"result"`
}

// CustomInput for testing
type CustomInput struct {
	Data string `json:"data"`
}

// CustomOutput for testing
type CustomOutput struct {
	Result string `json:"result"`
}

// TestService for testing
type TestService struct{}

func (s TestService) Hello(name string) (string, error) {
	return "Hello, " + name + "!", nil
}

func (s TestService) Create(user User) (string, error) {
	return "Created user " + user.Name, nil
}

func (s TestService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "Hello",
			HTTPMethod: "GET",
			InputType:  reflect.TypeOf(""),
			OutputType: reflect.TypeOf(""),
			Func:       reflect.ValueOf(s).MethodByName("Hello"),
		},
		{
			Name:       "Create",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(User{}),
			OutputType: reflect.TypeOf(""),
			Func:       reflect.ValueOf(s).MethodByName("Create"),
		},
	}
}

// InvalidMethodService for testing
type InvalidMethodService struct{}

func (s InvalidMethodService) InvalidMethod(input string) (string, error) {
	return "", fmt.Errorf("this method has an invalid HTTP method")
}

func (s InvalidMethodService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "InvalidMethod",
			HTTPMethod: "INVALID",
			InputType:  reflect.TypeOf(""),
			OutputType: reflect.TypeOf(""),
			Func:       reflect.ValueOf(s).MethodByName("InvalidMethod"),
		},
	}
}

// InvalidSigService for testing
type InvalidSigService struct{}

func (s InvalidSigService) BadMethod() string {
	return "this method has an invalid signature"
}

func (s InvalidSigService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "BadMethod",
			HTTPMethod: "GET",
			InputType:  nil,
			OutputType: reflect.TypeOf(""),
			Func:       reflect.ValueOf(s).MethodByName("BadMethod"),
		},
	}
}

// MultiMethodService for testing
type MultiMethodService struct{}

func (s MultiMethodService) GetMethod(name string) (MultiOutput, error) {
	if name == "error" {
		return MultiOutput{}, fmt.Errorf("simulated server error")
	}
	return MultiOutput{Result: "GET: " + name}, nil
}

func (s MultiMethodService) PostMethod(input MultiInput) (MultiOutput, error) {
	if input.Value == "error" {
		return MultiOutput{}, fmt.Errorf("simulated server error")
	}
	return MultiOutput{Result: "POST: " + input.Value}, nil
}

func (s MultiMethodService) PutMethod(input MultiInput) (MultiOutput, error) {
	if input.Value == "error" {
		return MultiOutput{}, fmt.Errorf("simulated server error")
	}
	return MultiOutput{Result: "PUT: " + input.Value}, nil
}

func (s MultiMethodService) DeleteMethod(input MultiInput) (MultiOutput, error) {
	if input.Value == "error" {
		return MultiOutput{}, fmt.Errorf("simulated server error")
	}
	return MultiOutput{Result: "DELETE: " + input.Value}, nil
}

func (s MultiMethodService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "GetMethod",
			HTTPMethod: "GET",
			InputType:  reflect.TypeOf(""),
			OutputType: reflect.TypeOf(MultiOutput{}),
			Func:       reflect.ValueOf(s).MethodByName("GetMethod"),
		},
		{
			Name:       "PostMethod",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(MultiInput{}),
			OutputType: reflect.TypeOf(MultiOutput{}),
			Func:       reflect.ValueOf(s).MethodByName("PostMethod"),
		},
		{
			Name:       "PutMethod",
			HTTPMethod: "PUT",
			InputType:  reflect.TypeOf(MultiInput{}),
			OutputType: reflect.TypeOf(MultiOutput{}),
			Func:       reflect.ValueOf(s).MethodByName("PutMethod"),
		},
		{
			Name:       "DeleteMethod",
			HTTPMethod: "DELETE",
			InputType:  reflect.TypeOf(MultiInput{}),
			OutputType: reflect.TypeOf(MultiOutput{}),
			Func:       reflect.ValueOf(s).MethodByName("DeleteMethod"),
		},
	}
}

// CustomPathService for testing
type CustomPathService struct{}

func (s CustomPathService) Process(input CustomInput) (CustomOutput, error) {
	return CustomOutput{Result: "Processed: " + input.Data}, nil
}

func (s CustomPathService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "Process",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(CustomInput{}),
			OutputType: reflect.TypeOf(CustomOutput{}),
			Func:       reflect.ValueOf(s).MethodByName("Process"),
		},
	}
}
