//
//Copyright 2023 The Kapacity Authors.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.20.3
// source: provider.proto

package api

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ProviderService_QueryLatest_FullMethodName = "/io.kapacitystack.metric.ProviderService/QueryLatest"
	ProviderService_Query_FullMethodName       = "/io.kapacitystack.metric.ProviderService/Query"
)

// ProviderServiceClient is the client API for ProviderService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProviderServiceClient interface {
	QueryLatest(ctx context.Context, in *QueryLatestRequest, opts ...grpc.CallOption) (*QueryLatestResponse, error)
	Query(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*QueryResponse, error)
}

type providerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewProviderServiceClient(cc grpc.ClientConnInterface) ProviderServiceClient {
	return &providerServiceClient{cc}
}

func (c *providerServiceClient) QueryLatest(ctx context.Context, in *QueryLatestRequest, opts ...grpc.CallOption) (*QueryLatestResponse, error) {
	out := new(QueryLatestResponse)
	err := c.cc.Invoke(ctx, ProviderService_QueryLatest_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerServiceClient) Query(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*QueryResponse, error) {
	out := new(QueryResponse)
	err := c.cc.Invoke(ctx, ProviderService_Query_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProviderServiceServer is the server API for ProviderService service.
// All implementations must embed UnimplementedProviderServiceServer
// for forward compatibility
type ProviderServiceServer interface {
	QueryLatest(context.Context, *QueryLatestRequest) (*QueryLatestResponse, error)
	Query(context.Context, *QueryRequest) (*QueryResponse, error)
	mustEmbedUnimplementedProviderServiceServer()
}

// UnimplementedProviderServiceServer must be embedded to have forward compatible implementations.
type UnimplementedProviderServiceServer struct {
}

func (UnimplementedProviderServiceServer) QueryLatest(context.Context, *QueryLatestRequest) (*QueryLatestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryLatest not implemented")
}
func (UnimplementedProviderServiceServer) Query(context.Context, *QueryRequest) (*QueryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Query not implemented")
}
func (UnimplementedProviderServiceServer) mustEmbedUnimplementedProviderServiceServer() {}

// UnsafeProviderServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProviderServiceServer will
// result in compilation errors.
type UnsafeProviderServiceServer interface {
	mustEmbedUnimplementedProviderServiceServer()
}

func RegisterProviderServiceServer(s grpc.ServiceRegistrar, srv ProviderServiceServer) {
	s.RegisterService(&ProviderService_ServiceDesc, srv)
}

func _ProviderService_QueryLatest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryLatestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderServiceServer).QueryLatest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderService_QueryLatest_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderServiceServer).QueryLatest(ctx, req.(*QueryLatestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderService_Query_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderServiceServer).Query(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderService_Query_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderServiceServer).Query(ctx, req.(*QueryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ProviderService_ServiceDesc is the grpc.ServiceDesc for ProviderService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ProviderService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "io.kapacitystack.metric.ProviderService",
	HandlerType: (*ProviderServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "QueryLatest",
			Handler:    _ProviderService_QueryLatest_Handler,
		},
		{
			MethodName: "Query",
			Handler:    _ProviderService_Query_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "provider.proto",
}
