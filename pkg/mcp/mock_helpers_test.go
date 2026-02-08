package mcp

import (
	"context"

	"go.uber.org/mock/gomock"
)

// setupMockAgentClient configures a generated MockAgentClient with common defaults
// that match the behavior of the old hand-rolled mock constructor.
// CallTool is NOT configured - use setupMockAgentClientWithCallTool or set it yourself.
func setupMockAgentClient(ctrl *gomock.Controller, name string, tools []Tool) *MockAgentClient {
	mock := NewMockAgentClient(ctrl)
	mock.EXPECT().Name().Return(name).AnyTimes()
	mock.EXPECT().Tools().Return(tools).AnyTimes()
	mock.EXPECT().IsInitialized().Return(true).AnyTimes()
	mock.EXPECT().ServerInfo().Return(ServerInfo{Name: name, Version: "1.0.0"}).AnyTimes()
	mock.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
	mock.EXPECT().RefreshTools(gomock.Any()).Return(nil).AnyTimes()
	return mock
}

// setupMockAgentClientWithCallTool is like setupMockAgentClient but also sets a
// default CallTool implementation that returns "mock result for <tool>".
func setupMockAgentClientWithCallTool(ctrl *gomock.Controller, name string, tools []Tool) *MockAgentClient {
	mock := setupMockAgentClient(ctrl, name, tools)
	mock.EXPECT().CallTool(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, toolName string, arguments map[string]any) (*ToolCallResult, error) {
			return &ToolCallResult{
				Content: []Content{NewTextContent("mock result for " + toolName)},
			}, nil
		},
	).AnyTimes()
	return mock
}
