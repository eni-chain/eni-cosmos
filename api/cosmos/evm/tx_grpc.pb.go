// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: goeni/evm/tx.proto

package evm

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
	Msg_EVMTransaction_FullMethodName           = "/goeni.evm.Msg/EVMTransaction"
	Msg_Send_FullMethodName                     = "/goeni.evm.Msg/Send"
	Msg_RegisterPointer_FullMethodName          = "/goeni.evm.Msg/RegisterPointer"
	Msg_AssociateContractAddress_FullMethodName = "/goeni.evm.Msg/AssociateContractAddress"
	Msg_Associate_FullMethodName                = "/goeni.evm.Msg/Associate"
)

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MsgClient interface {
	EVMTransaction(ctx context.Context, in *MsgEVMTransaction, opts ...grpc.CallOption) (*MsgEVMTransactionResponse, error)
	Send(ctx context.Context, in *MsgSend, opts ...grpc.CallOption) (*MsgSendResponse, error)
	RegisterPointer(ctx context.Context, in *MsgRegisterPointer, opts ...grpc.CallOption) (*MsgRegisterPointerResponse, error)
	AssociateContractAddress(ctx context.Context, in *MsgAssociateContractAddress, opts ...grpc.CallOption) (*MsgAssociateContractAddressResponse, error)
	Associate(ctx context.Context, in *MsgAssociate, opts ...grpc.CallOption) (*MsgAssociateResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) EVMTransaction(ctx context.Context, in *MsgEVMTransaction, opts ...grpc.CallOption) (*MsgEVMTransactionResponse, error) {
	out := new(MsgEVMTransactionResponse)
	err := c.cc.Invoke(ctx, Msg_EVMTransaction_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) Send(ctx context.Context, in *MsgSend, opts ...grpc.CallOption) (*MsgSendResponse, error) {
	out := new(MsgSendResponse)
	err := c.cc.Invoke(ctx, Msg_Send_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) RegisterPointer(ctx context.Context, in *MsgRegisterPointer, opts ...grpc.CallOption) (*MsgRegisterPointerResponse, error) {
	out := new(MsgRegisterPointerResponse)
	err := c.cc.Invoke(ctx, Msg_RegisterPointer_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) AssociateContractAddress(ctx context.Context, in *MsgAssociateContractAddress, opts ...grpc.CallOption) (*MsgAssociateContractAddressResponse, error) {
	out := new(MsgAssociateContractAddressResponse)
	err := c.cc.Invoke(ctx, Msg_AssociateContractAddress_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) Associate(ctx context.Context, in *MsgAssociate, opts ...grpc.CallOption) (*MsgAssociateResponse, error) {
	out := new(MsgAssociateResponse)
	err := c.cc.Invoke(ctx, Msg_Associate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
// All implementations must embed UnimplementedMsgServer
// for forward compatibility
type MsgServer interface {
	EVMTransaction(context.Context, *MsgEVMTransaction) (*MsgEVMTransactionResponse, error)
	Send(context.Context, *MsgSend) (*MsgSendResponse, error)
	RegisterPointer(context.Context, *MsgRegisterPointer) (*MsgRegisterPointerResponse, error)
	AssociateContractAddress(context.Context, *MsgAssociateContractAddress) (*MsgAssociateContractAddressResponse, error)
	Associate(context.Context, *MsgAssociate) (*MsgAssociateResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) EVMTransaction(context.Context, *MsgEVMTransaction) (*MsgEVMTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EVMTransaction not implemented")
}
func (UnimplementedMsgServer) Send(context.Context, *MsgSend) (*MsgSendResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Send not implemented")
}
func (UnimplementedMsgServer) RegisterPointer(context.Context, *MsgRegisterPointer) (*MsgRegisterPointerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterPointer not implemented")
}
func (UnimplementedMsgServer) AssociateContractAddress(context.Context, *MsgAssociateContractAddress) (*MsgAssociateContractAddressResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AssociateContractAddress not implemented")
}
func (UnimplementedMsgServer) Associate(context.Context, *MsgAssociate) (*MsgAssociateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Associate not implemented")
}
func (UnimplementedMsgServer) mustEmbedUnimplementedMsgServer() {}

// UnsafeMsgServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MsgServer will
// result in compilation errors.
type UnsafeMsgServer interface {
	mustEmbedUnimplementedMsgServer()
}

func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&Msg_ServiceDesc, srv)
}

func _Msg_EVMTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEVMTransaction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).EVMTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_EVMTransaction_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).EVMTransaction(ctx, req.(*MsgEVMTransaction))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_Send_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSend)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).Send(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_Send_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).Send(ctx, req.(*MsgSend))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_RegisterPointer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterPointer)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterPointer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_RegisterPointer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterPointer(ctx, req.(*MsgRegisterPointer))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_AssociateContractAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAssociateContractAddress)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AssociateContractAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_AssociateContractAddress_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AssociateContractAddress(ctx, req.(*MsgAssociateContractAddress))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_Associate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAssociate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).Associate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_Associate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).Associate(ctx, req.(*MsgAssociate))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "goeni.evm.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "EVMTransaction",
			Handler:    _Msg_EVMTransaction_Handler,
		},
		{
			MethodName: "Send",
			Handler:    _Msg_Send_Handler,
		},
		{
			MethodName: "RegisterPointer",
			Handler:    _Msg_RegisterPointer_Handler,
		},
		{
			MethodName: "AssociateContractAddress",
			Handler:    _Msg_AssociateContractAddress_Handler,
		},
		{
			MethodName: "Associate",
			Handler:    _Msg_Associate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "goeni/evm/tx.proto",
}
