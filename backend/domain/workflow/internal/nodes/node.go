/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nodes

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	schema2 "github.com/coze-dev/coze-studio/backend/domain/workflow/internal/schema"
)

type InvokableNodeWOpt interface {
	Invoke(ctx context.Context, in map[string]any, opts ...NodeOption) (
		map[string]any, error)
}

type InvokableNode interface {
	Invoke(ctx context.Context, in map[string]any) (
		map[string]any, error)
}

type StreamableNodeWOpt interface {
	Stream(ctx context.Context, in map[string]any, opts ...NodeOption) (
		*schema.StreamReader[map[string]any], error)
}

type StreamableNode interface {
	Stream(ctx context.Context, in map[string]any) (
		*schema.StreamReader[map[string]any], error)
}

type CollectableNodeWOpt interface {
	Collect(ctx context.Context, in *schema.StreamReader[map[string]any], opts ...NodeOption) (
		map[string]any, error)
}

type CollectableNode interface {
	Collect(ctx context.Context, in *schema.StreamReader[map[string]any]) (
		map[string]any, error)
}

type TransformableNodeWOpt interface {
	Transform(ctx context.Context, in *schema.StreamReader[map[string]any], opts ...NodeOption) (
		*schema.StreamReader[map[string]any], error)
}

type TransformableNode interface {
	Transform(ctx context.Context, in *schema.StreamReader[map[string]any]) (
		*schema.StreamReader[map[string]any], error)
}

type CallbackInputConverted interface {
	ToCallbackInput(ctx context.Context, in map[string]any) (map[string]any, error)
}

type CallbackOutputConverted interface {
	ToCallbackOutput(ctx context.Context, out map[string]any) (*StructuredCallbackOutput, error)
}

type Initializer interface {
	Init(ctx context.Context) (context.Context, error)
}

type AdaptOptions struct {
	Canvas *vo.Canvas
}

type AdaptOption func(*AdaptOptions)

func WithCanvas(canvas *vo.Canvas) AdaptOption {
	return func(opts *AdaptOptions) {
		opts.Canvas = canvas
	}
}

func GetAdaptOptions(opts ...AdaptOption) *AdaptOptions {
	options := &AdaptOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

type NodeAdaptor interface {
	Adapt(ctx context.Context, n *vo.Node, opts ...AdaptOption) (*schema2.NodeSchema, error)
}

var (
	adaptors = map[entity.NodeType]func() NodeAdaptor{}
)

func RegisterAdaptor(et entity.NodeType, f func() NodeAdaptor) {
	adaptors[et] = f
}

func GetAdaptor(et entity.NodeType) (NodeAdaptor, bool) {
	na, ok := adaptors[et]
	if !ok {
		panic(fmt.Sprintf("node type %s not registered", et))
	}
	return na(), ok
}
