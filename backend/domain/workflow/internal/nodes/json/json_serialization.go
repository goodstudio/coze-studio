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

package json

import (
	"context"
	"fmt"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/canvas/convert"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/schema"
	"github.com/coze-dev/coze-studio/backend/pkg/sonic"
)

const (
	InputKeySerialization  = "input"
	OutputKeySerialization = "output"
)

type SerializationConfig struct{}

func (s *SerializationConfig) Adapt(_ context.Context, n *vo.Node, _ ...nodes.AdaptOption) (*schema.NodeSchema, error) {
	ns := &schema.NodeSchema{
		Key:     vo.NodeKey(n.ID),
		Type:    entity.NodeTypeJsonSerialization,
		Name:    n.Data.Meta.Title,
		Configs: s,
	}

	if err := convert.SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := convert.SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func (s *SerializationConfig) Build(_ context.Context, _ *schema.NodeSchema, _ ...schema.BuildOption) (
	any, error) {
	return &Serializer{}, nil
}

type Serializer struct{}

func (js *Serializer) Invoke(_ context.Context, input map[string]any) (map[string]any, error) {
	// Directly use the input map for serialization
	if input == nil {
		return nil, fmt.Errorf("input data for serialization cannot be nil")
	}

	originData := input[InputKeySerialization]
	serializedData, err := sonic.Marshal(originData) // Serialize the entire input map
	if err != nil {
		return nil, fmt.Errorf("serialization error: %w", err)
	}
	return map[string]any{OutputKeySerialization: string(serializedData)}, nil
}
