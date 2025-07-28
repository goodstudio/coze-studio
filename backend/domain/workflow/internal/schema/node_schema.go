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

package schema

import (
	"github.com/cloudwego/eino/compose"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
)

type NodeSchema struct {
	Key  vo.NodeKey `json:"key"`
	Name string     `json:"name"`

	Type entity.NodeType `json:"type"`

	// Configs are node specific configurations with pre-defined config key and config value.
	// Will not participate in request-time field mapping, nor as node's static values.
	// In a word, these Configs are INTERNAL to node's implementation, the workflow layer is not aware of them.
	Configs any `json:"configs,omitempty"`

	InputTypes   map[string]*vo.TypeInfo `json:"input_types,omitempty"`
	InputSources []*vo.FieldInfo         `json:"input_sources,omitempty"`

	OutputTypes   map[string]*vo.TypeInfo `json:"output_types,omitempty"`
	OutputSources []*vo.FieldInfo         `json:"output_sources,omitempty"` // only applicable to composite nodes such as Batch or Loop

	ExceptionConfigs *ExceptionConfig `json:"exception_configs,omitempty"` // generic configurations applicable to most nodes
	StreamConfigs    *StreamConfig    `json:"stream_configs,omitempty"`

	SubWorkflowBasic  *entity.WorkflowBasic `json:"sub_workflow_basic,omitempty"`
	SubWorkflowSchema *WorkflowSchema       `json:"sub_workflow_schema,omitempty"`

	FullSources map[string]*SourceInfo `json:"full_sources,omitempty"`

	Lambda *compose.Lambda // not serializable, used for internal test.
}

type RequireCheckpoint interface {
	RequireCheckpoint() bool
}

type ExceptionConfig struct {
	TimeoutMS   int64                `json:"timeout_ms,omitempty"`   // timeout in milliseconds, 0 means no timeout
	MaxRetry    int64                `json:"max_retry,omitempty"`    // max retry times, 0 means no retry
	ProcessType *vo.ErrorProcessType `json:"process_type,omitempty"` // error process type, 0 means throw error
	DataOnErr   string               `json:"data_on_err,omitempty"`  // data to return when error, effective when ProcessType==Default occurs
}

type StreamConfig struct {
	// whether this node has the ability to produce genuine streaming output.
	// not include nodes that only passes stream down as they receives them
	CanGeneratesStream bool `json:"can_generates_stream,omitempty"`
	// whether this node prioritize streaming input over none-streaming input.
	// not include nodes that can accept both and does not have preference.
	RequireStreamingInput bool `json:"can_process_stream,omitempty"`
}

func (s *NodeSchema) SetConfigKV(key string, value any) {
	if s.Configs == nil {
		s.Configs = make(map[string]any)
	}

	s.Configs.(map[string]any)[key] = value
}

func (s *NodeSchema) SetInputType(key string, t *vo.TypeInfo) {
	if s.InputTypes == nil {
		s.InputTypes = make(map[string]*vo.TypeInfo)
	}
	s.InputTypes[key] = t
}

func (s *NodeSchema) AddInputSource(info ...*vo.FieldInfo) {
	s.InputSources = append(s.InputSources, info...)
}

func (s *NodeSchema) SetOutputType(key string, t *vo.TypeInfo) {
	if s.OutputTypes == nil {
		s.OutputTypes = make(map[string]*vo.TypeInfo)
	}
	s.OutputTypes[key] = t
}

func (s *NodeSchema) AddOutputSource(info ...*vo.FieldInfo) {
	s.OutputSources = append(s.OutputSources, info...)
}
