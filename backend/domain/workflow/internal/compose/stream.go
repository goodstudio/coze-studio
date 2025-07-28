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

package compose

import (
	"fmt"

	"github.com/cloudwego/eino/compose"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/schema"
)

// GetFullSources calculates REAL input sources for a node.
// It may be different from a NodeSchema's InputSources because of the following reasons:
//  1. a inner node under composite node may refer to a field from a node in its parent workflow,
//     this is instead routed to and sourced from the inner workflow's start node.
//  2. at the same time, the composite node needs to delegate the input source to the inner workflow.
//  3. also, some node may have implicit input sources not defined in its NodeSchema's InputSources.
func GetFullSources(s *schema.NodeSchema, allNS map[vo.NodeKey]*schema.NodeSchema, dep *dependencyInfo) (
	map[string]*schema.SourceInfo, error) {
	fullSource := make(map[string]*schema.SourceInfo)
	var fieldInfos []vo.FieldInfo
	for _, s := range dep.staticValues {
		fieldInfos = append(fieldInfos, vo.FieldInfo{
			Path:   s.path,
			Source: vo.FieldSource{Val: s.val},
		})
	}

	for _, v := range dep.variableInfos {
		fieldInfos = append(fieldInfos, vo.FieldInfo{
			Path: v.toPath,
			Source: vo.FieldSource{
				Ref: &vo.Reference{
					VariableType: &v.varType,
					FromPath:     v.fromPath[1:],
				},
			},
		})
	}

	for f := range dep.inputsFull {
		fieldInfos = append(fieldInfos, vo.FieldInfo{
			Path: []string{""},
			Source: vo.FieldSource{Ref: &vo.Reference{
				FromNodeKey: f,
				FromPath:    []string{""},
			}},
		})
	}

	for f, ms := range dep.inputs {
		for _, m := range ms {
			fieldInfos = append(fieldInfos, vo.FieldInfo{
				Path: m.ToPath(),
				Source: vo.FieldSource{Ref: &vo.Reference{
					FromNodeKey: f,
					FromPath:    m.FromPath(),
				}},
			})
		}
	}

	for f := range dep.inputsNoDirectDependencyFull {
		fieldInfos = append(fieldInfos, vo.FieldInfo{
			Path: []string{""},
			Source: vo.FieldSource{Ref: &vo.Reference{
				FromNodeKey: f,
				FromPath:    []string{""},
			}},
		})
	}

	for f, ms := range dep.inputsNoDirectDependency {
		for _, m := range ms {
			fieldInfos = append(fieldInfos, vo.FieldInfo{
				Path: m.ToPath(),
				Source: vo.FieldSource{Ref: &vo.Reference{
					FromNodeKey: f,
					FromPath:    m.FromPath(),
				}},
			})
		}
	}

	for i := range fieldInfos {
		fInfo := fieldInfos[i]
		path := fInfo.Path
		currentSource := fullSource
		var (
			tInfo    *vo.TypeInfo
			lastPath string
		)
		if len(path) > 1 {
			tInfo = s.InputTypes[path[0]]
			for j := 0; j < len(path)-1; j++ {
				if j > 0 {
					tInfo = tInfo.Properties[path[j]]
				}
				if current, ok := currentSource[path[j]]; !ok {
					currentSource[path[j]] = &schema.SourceInfo{
						IsIntermediate: true,
						FieldType:      schema.FieldNotStream,
						TypeInfo:       tInfo,
						SubSources:     make(map[string]*schema.SourceInfo),
					}
				} else if !current.IsIntermediate {
					return nil, fmt.Errorf("existing sourceInfo for path %s is not intermediate, conflict", path[:j+1])
				}

				currentSource = currentSource[path[j]].SubSources
			}

			lastPath = path[len(path)-1]
			tInfo = tInfo.Properties[lastPath]
		} else {
			lastPath = path[0]
			tInfo = s.InputTypes[lastPath]
		}

		// static values or variables
		if fInfo.Source.Ref == nil || fInfo.Source.Ref.FromNodeKey == "" {
			currentSource[lastPath] = &schema.SourceInfo{
				IsIntermediate: false,
				FieldType:      schema.FieldNotStream,
				TypeInfo:       tInfo,
			}
			continue
		}

		fromNodeKey := fInfo.Source.Ref.FromNodeKey
		var (
			streamType schema.FieldStreamType
			err        error
		)
		if len(fromNodeKey) > 0 {
			if fromNodeKey == compose.START {
				streamType = schema.FieldNotStream // TODO: set start node to not stream for now until composite node supports transform
			} else {
				fromNode, ok := allNS[fromNodeKey]
				if !ok {
					return nil, fmt.Errorf("node %s not found", fromNodeKey)
				}
				streamType, err = IsStreamingField(fromNode, fInfo.Source.Ref.FromPath, allNS)
				if err != nil {
					return nil, err
				}
			}
		}

		currentSource[lastPath] = &schema.SourceInfo{
			IsIntermediate: false,
			FieldType:      streamType,
			FromNodeKey:    fromNodeKey,
			FromPath:       fInfo.Source.Ref.FromPath,
			TypeInfo:       tInfo,
		}
	}

	return fullSource, nil
}

func IsStreamingField(s *schema.NodeSchema, path compose.FieldPath, allNS map[vo.NodeKey]*schema.NodeSchema) (schema.FieldStreamType, error) {
	if s.Type == entity.NodeTypeExit {
		if s.StreamConfigs.RequireStreamingInput {
			if len(path) == 1 && path[0] == "output" {
				return schema.FieldIsStream, nil
			}
		}

		return schema.FieldNotStream, nil
	} else if s.Type == entity.NodeTypeSubWorkflow { // TODO: why not use sub workflow's Mode configuration directly?
		subSC := s.SubWorkflowSchema
		subExit := subSC.GetNode(entity.ExitNodeKey)
		subStreamType, err := IsStreamingField(subExit, path, nil)
		if err != nil {
			return schema.FieldNotStream, err
		}

		return subStreamType, nil
	} else if s.Type == entity.NodeTypeVariableAggregator {
		if len(path) == 2 { // asking about a specific index within a group
			for _, fInfo := range s.InputSources {
				if len(fInfo.Path) == len(path) {
					equal := true
					for i := range fInfo.Path {
						if fInfo.Path[i] != path[i] {
							equal = false
							break
						}
					}

					if equal {
						if fInfo.Source.Ref == nil || fInfo.Source.Ref.FromNodeKey == "" {
							return schema.FieldNotStream, nil
						}
						fromNodeKey := fInfo.Source.Ref.FromNodeKey
						fromNode, ok := allNS[fromNodeKey]
						if !ok {
							return schema.FieldNotStream, fmt.Errorf("node %s not found", fromNodeKey)
						}
						return IsStreamingField(fromNode, fInfo.Source.Ref.FromPath, allNS)
					}
				}
			}
		} else if len(path) == 1 { // asking about the entire group
			var streamCount, notStreamCount int
			for _, fInfo := range s.InputSources {
				if fInfo.Path[0] == path[0] { // belong to the group
					if fInfo.Source.Ref != nil && len(fInfo.Source.Ref.FromNodeKey) > 0 {
						fromNode, ok := allNS[fInfo.Source.Ref.FromNodeKey]
						if !ok {
							return schema.FieldNotStream, fmt.Errorf("node %s not found", fInfo.Source.Ref.FromNodeKey)
						}
						subStreamType, err := IsStreamingField(fromNode, fInfo.Source.Ref.FromPath, allNS)
						if err != nil {
							return schema.FieldNotStream, err
						}

						if subStreamType == schema.FieldMaybeStream {
							return schema.FieldMaybeStream, nil
						} else if subStreamType == schema.FieldIsStream {
							streamCount++
						} else {
							notStreamCount++
						}
					}
				}
			}

			if streamCount > 0 && notStreamCount == 0 {
				return schema.FieldIsStream, nil
			}

			if streamCount == 0 && notStreamCount > 0 {
				return schema.FieldNotStream, nil
			}

			return schema.FieldMaybeStream, nil
		}
	}

	if s.Type != entity.NodeTypeLLM {
		return schema.FieldNotStream, nil
	}

	if len(path) != 1 {
		return schema.FieldNotStream, nil
	}

	outputs := s.OutputTypes
	if len(outputs) != 1 && len(outputs) != 2 {
		return schema.FieldNotStream, nil
	}

	var outputKey string
	for key, output := range outputs {
		if output.Type != vo.DataTypeString {
			return schema.FieldNotStream, nil
		}

		if key != "reasoning_content" {
			if len(outputKey) > 0 {
				return schema.FieldNotStream, nil
			}
			outputKey = key
		}
	}

	field := path[0]
	if field == "reasoning_content" || field == outputKey {
		return schema.FieldIsStream, nil
	}

	return schema.FieldNotStream, nil
}
