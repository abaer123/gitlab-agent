// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/gitaly/proto/go/gitalypb (interfaces: CommitServiceClient,CommitService_TreeEntryClient)

// Package mock_gitaly is a generated GoMock package.
package mock_gitaly

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	gitalypb "gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// MockCommitServiceClient is a mock of CommitServiceClient interface.
type MockCommitServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockCommitServiceClientMockRecorder
}

// MockCommitServiceClientMockRecorder is the mock recorder for MockCommitServiceClient.
type MockCommitServiceClientMockRecorder struct {
	mock *MockCommitServiceClient
}

// NewMockCommitServiceClient creates a new mock instance.
func NewMockCommitServiceClient(ctrl *gomock.Controller) *MockCommitServiceClient {
	mock := &MockCommitServiceClient{ctrl: ctrl}
	mock.recorder = &MockCommitServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCommitServiceClient) EXPECT() *MockCommitServiceClientMockRecorder {
	return m.recorder
}

// CommitIsAncestor mocks base method.
func (m *MockCommitServiceClient) CommitIsAncestor(arg0 context.Context, arg1 *gitalypb.CommitIsAncestorRequest, arg2 ...grpc.CallOption) (*gitalypb.CommitIsAncestorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CommitIsAncestor", varargs...)
	ret0, _ := ret[0].(*gitalypb.CommitIsAncestorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CommitIsAncestor indicates an expected call of CommitIsAncestor.
func (mr *MockCommitServiceClientMockRecorder) CommitIsAncestor(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitIsAncestor", reflect.TypeOf((*MockCommitServiceClient)(nil).CommitIsAncestor), varargs...)
}

// CommitLanguages mocks base method.
func (m *MockCommitServiceClient) CommitLanguages(arg0 context.Context, arg1 *gitalypb.CommitLanguagesRequest, arg2 ...grpc.CallOption) (*gitalypb.CommitLanguagesResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CommitLanguages", varargs...)
	ret0, _ := ret[0].(*gitalypb.CommitLanguagesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CommitLanguages indicates an expected call of CommitLanguages.
func (mr *MockCommitServiceClientMockRecorder) CommitLanguages(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitLanguages", reflect.TypeOf((*MockCommitServiceClient)(nil).CommitLanguages), varargs...)
}

// CommitStats mocks base method.
func (m *MockCommitServiceClient) CommitStats(arg0 context.Context, arg1 *gitalypb.CommitStatsRequest, arg2 ...grpc.CallOption) (*gitalypb.CommitStatsResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CommitStats", varargs...)
	ret0, _ := ret[0].(*gitalypb.CommitStatsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CommitStats indicates an expected call of CommitStats.
func (mr *MockCommitServiceClientMockRecorder) CommitStats(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitStats", reflect.TypeOf((*MockCommitServiceClient)(nil).CommitStats), varargs...)
}

// CommitsBetween mocks base method.
func (m *MockCommitServiceClient) CommitsBetween(arg0 context.Context, arg1 *gitalypb.CommitsBetweenRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_CommitsBetweenClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CommitsBetween", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_CommitsBetweenClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CommitsBetween indicates an expected call of CommitsBetween.
func (mr *MockCommitServiceClientMockRecorder) CommitsBetween(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitsBetween", reflect.TypeOf((*MockCommitServiceClient)(nil).CommitsBetween), varargs...)
}

// CommitsByMessage mocks base method.
func (m *MockCommitServiceClient) CommitsByMessage(arg0 context.Context, arg1 *gitalypb.CommitsByMessageRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_CommitsByMessageClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CommitsByMessage", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_CommitsByMessageClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CommitsByMessage indicates an expected call of CommitsByMessage.
func (mr *MockCommitServiceClientMockRecorder) CommitsByMessage(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitsByMessage", reflect.TypeOf((*MockCommitServiceClient)(nil).CommitsByMessage), varargs...)
}

// CountCommits mocks base method.
func (m *MockCommitServiceClient) CountCommits(arg0 context.Context, arg1 *gitalypb.CountCommitsRequest, arg2 ...grpc.CallOption) (*gitalypb.CountCommitsResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CountCommits", varargs...)
	ret0, _ := ret[0].(*gitalypb.CountCommitsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CountCommits indicates an expected call of CountCommits.
func (mr *MockCommitServiceClientMockRecorder) CountCommits(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CountCommits", reflect.TypeOf((*MockCommitServiceClient)(nil).CountCommits), varargs...)
}

// CountDivergingCommits mocks base method.
func (m *MockCommitServiceClient) CountDivergingCommits(arg0 context.Context, arg1 *gitalypb.CountDivergingCommitsRequest, arg2 ...grpc.CallOption) (*gitalypb.CountDivergingCommitsResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CountDivergingCommits", varargs...)
	ret0, _ := ret[0].(*gitalypb.CountDivergingCommitsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CountDivergingCommits indicates an expected call of CountDivergingCommits.
func (mr *MockCommitServiceClientMockRecorder) CountDivergingCommits(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CountDivergingCommits", reflect.TypeOf((*MockCommitServiceClient)(nil).CountDivergingCommits), varargs...)
}

// FilterShasWithSignatures mocks base method.
func (m *MockCommitServiceClient) FilterShasWithSignatures(arg0 context.Context, arg1 ...grpc.CallOption) (gitalypb.CommitService_FilterShasWithSignaturesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "FilterShasWithSignatures", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_FilterShasWithSignaturesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FilterShasWithSignatures indicates an expected call of FilterShasWithSignatures.
func (mr *MockCommitServiceClientMockRecorder) FilterShasWithSignatures(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FilterShasWithSignatures", reflect.TypeOf((*MockCommitServiceClient)(nil).FilterShasWithSignatures), varargs...)
}

// FindAllCommits mocks base method.
func (m *MockCommitServiceClient) FindAllCommits(arg0 context.Context, arg1 *gitalypb.FindAllCommitsRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_FindAllCommitsClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "FindAllCommits", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_FindAllCommitsClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindAllCommits indicates an expected call of FindAllCommits.
func (mr *MockCommitServiceClientMockRecorder) FindAllCommits(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindAllCommits", reflect.TypeOf((*MockCommitServiceClient)(nil).FindAllCommits), varargs...)
}

// FindCommit mocks base method.
func (m *MockCommitServiceClient) FindCommit(arg0 context.Context, arg1 *gitalypb.FindCommitRequest, arg2 ...grpc.CallOption) (*gitalypb.FindCommitResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "FindCommit", varargs...)
	ret0, _ := ret[0].(*gitalypb.FindCommitResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindCommit indicates an expected call of FindCommit.
func (mr *MockCommitServiceClientMockRecorder) FindCommit(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindCommit", reflect.TypeOf((*MockCommitServiceClient)(nil).FindCommit), varargs...)
}

// FindCommits mocks base method.
func (m *MockCommitServiceClient) FindCommits(arg0 context.Context, arg1 *gitalypb.FindCommitsRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_FindCommitsClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "FindCommits", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_FindCommitsClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindCommits indicates an expected call of FindCommits.
func (mr *MockCommitServiceClientMockRecorder) FindCommits(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindCommits", reflect.TypeOf((*MockCommitServiceClient)(nil).FindCommits), varargs...)
}

// GetCommitMessages mocks base method.
func (m *MockCommitServiceClient) GetCommitMessages(arg0 context.Context, arg1 *gitalypb.GetCommitMessagesRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_GetCommitMessagesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetCommitMessages", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_GetCommitMessagesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCommitMessages indicates an expected call of GetCommitMessages.
func (mr *MockCommitServiceClientMockRecorder) GetCommitMessages(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCommitMessages", reflect.TypeOf((*MockCommitServiceClient)(nil).GetCommitMessages), varargs...)
}

// GetCommitSignatures mocks base method.
func (m *MockCommitServiceClient) GetCommitSignatures(arg0 context.Context, arg1 *gitalypb.GetCommitSignaturesRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_GetCommitSignaturesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetCommitSignatures", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_GetCommitSignaturesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCommitSignatures indicates an expected call of GetCommitSignatures.
func (mr *MockCommitServiceClientMockRecorder) GetCommitSignatures(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCommitSignatures", reflect.TypeOf((*MockCommitServiceClient)(nil).GetCommitSignatures), varargs...)
}

// GetTreeEntries mocks base method.
func (m *MockCommitServiceClient) GetTreeEntries(arg0 context.Context, arg1 *gitalypb.GetTreeEntriesRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_GetTreeEntriesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetTreeEntries", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_GetTreeEntriesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTreeEntries indicates an expected call of GetTreeEntries.
func (mr *MockCommitServiceClientMockRecorder) GetTreeEntries(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTreeEntries", reflect.TypeOf((*MockCommitServiceClient)(nil).GetTreeEntries), varargs...)
}

// LastCommitForPath mocks base method.
func (m *MockCommitServiceClient) LastCommitForPath(arg0 context.Context, arg1 *gitalypb.LastCommitForPathRequest, arg2 ...grpc.CallOption) (*gitalypb.LastCommitForPathResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "LastCommitForPath", varargs...)
	ret0, _ := ret[0].(*gitalypb.LastCommitForPathResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LastCommitForPath indicates an expected call of LastCommitForPath.
func (mr *MockCommitServiceClientMockRecorder) LastCommitForPath(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastCommitForPath", reflect.TypeOf((*MockCommitServiceClient)(nil).LastCommitForPath), varargs...)
}

// ListCommitsByOid mocks base method.
func (m *MockCommitServiceClient) ListCommitsByOid(arg0 context.Context, arg1 *gitalypb.ListCommitsByOidRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_ListCommitsByOidClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListCommitsByOid", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_ListCommitsByOidClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCommitsByOid indicates an expected call of ListCommitsByOid.
func (mr *MockCommitServiceClientMockRecorder) ListCommitsByOid(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCommitsByOid", reflect.TypeOf((*MockCommitServiceClient)(nil).ListCommitsByOid), varargs...)
}

// ListCommitsByRefName mocks base method.
func (m *MockCommitServiceClient) ListCommitsByRefName(arg0 context.Context, arg1 *gitalypb.ListCommitsByRefNameRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_ListCommitsByRefNameClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListCommitsByRefName", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_ListCommitsByRefNameClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCommitsByRefName indicates an expected call of ListCommitsByRefName.
func (mr *MockCommitServiceClientMockRecorder) ListCommitsByRefName(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCommitsByRefName", reflect.TypeOf((*MockCommitServiceClient)(nil).ListCommitsByRefName), varargs...)
}

// ListFiles mocks base method.
func (m *MockCommitServiceClient) ListFiles(arg0 context.Context, arg1 *gitalypb.ListFilesRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_ListFilesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListFiles", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_ListFilesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListFiles indicates an expected call of ListFiles.
func (mr *MockCommitServiceClientMockRecorder) ListFiles(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFiles", reflect.TypeOf((*MockCommitServiceClient)(nil).ListFiles), varargs...)
}

// ListLastCommitsForTree mocks base method.
func (m *MockCommitServiceClient) ListLastCommitsForTree(arg0 context.Context, arg1 *gitalypb.ListLastCommitsForTreeRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_ListLastCommitsForTreeClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListLastCommitsForTree", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_ListLastCommitsForTreeClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLastCommitsForTree indicates an expected call of ListLastCommitsForTree.
func (mr *MockCommitServiceClientMockRecorder) ListLastCommitsForTree(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLastCommitsForTree", reflect.TypeOf((*MockCommitServiceClient)(nil).ListLastCommitsForTree), varargs...)
}

// RawBlame mocks base method.
func (m *MockCommitServiceClient) RawBlame(arg0 context.Context, arg1 *gitalypb.RawBlameRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_RawBlameClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RawBlame", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_RawBlameClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RawBlame indicates an expected call of RawBlame.
func (mr *MockCommitServiceClientMockRecorder) RawBlame(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RawBlame", reflect.TypeOf((*MockCommitServiceClient)(nil).RawBlame), varargs...)
}

// TreeEntry mocks base method.
func (m *MockCommitServiceClient) TreeEntry(arg0 context.Context, arg1 *gitalypb.TreeEntryRequest, arg2 ...grpc.CallOption) (gitalypb.CommitService_TreeEntryClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "TreeEntry", varargs...)
	ret0, _ := ret[0].(gitalypb.CommitService_TreeEntryClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TreeEntry indicates an expected call of TreeEntry.
func (mr *MockCommitServiceClientMockRecorder) TreeEntry(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TreeEntry", reflect.TypeOf((*MockCommitServiceClient)(nil).TreeEntry), varargs...)
}

// MockCommitService_TreeEntryClient is a mock of CommitService_TreeEntryClient interface.
type MockCommitService_TreeEntryClient struct {
	ctrl     *gomock.Controller
	recorder *MockCommitService_TreeEntryClientMockRecorder
}

// MockCommitService_TreeEntryClientMockRecorder is the mock recorder for MockCommitService_TreeEntryClient.
type MockCommitService_TreeEntryClientMockRecorder struct {
	mock *MockCommitService_TreeEntryClient
}

// NewMockCommitService_TreeEntryClient creates a new mock instance.
func NewMockCommitService_TreeEntryClient(ctrl *gomock.Controller) *MockCommitService_TreeEntryClient {
	mock := &MockCommitService_TreeEntryClient{ctrl: ctrl}
	mock.recorder = &MockCommitService_TreeEntryClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCommitService_TreeEntryClient) EXPECT() *MockCommitService_TreeEntryClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockCommitService_TreeEntryClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockCommitService_TreeEntryClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockCommitService_TreeEntryClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockCommitService_TreeEntryClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockCommitService_TreeEntryClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockCommitService_TreeEntryClient)(nil).Context))
}

// Header mocks base method.
func (m *MockCommitService_TreeEntryClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockCommitService_TreeEntryClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockCommitService_TreeEntryClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockCommitService_TreeEntryClient) Recv() (*gitalypb.TreeEntryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*gitalypb.TreeEntryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockCommitService_TreeEntryClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockCommitService_TreeEntryClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m *MockCommitService_TreeEntryClient) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockCommitService_TreeEntryClientMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockCommitService_TreeEntryClient)(nil).RecvMsg), arg0)
}

// SendMsg mocks base method.
func (m *MockCommitService_TreeEntryClient) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockCommitService_TreeEntryClientMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockCommitService_TreeEntryClient)(nil).SendMsg), arg0)
}

// Trailer mocks base method.
func (m *MockCommitService_TreeEntryClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockCommitService_TreeEntryClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockCommitService_TreeEntryClient)(nil).Trailer))
}
