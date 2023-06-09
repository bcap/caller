package plan

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Execution is simply an ordered list of steps
type Execution []Step

func (e Execution) MarshalYAML() (interface{}, error) {
	return e.toMarshallable(), nil
}

func (e *Execution) UnmarshalYAML(node *yaml.Node) error {
	steps := make([]Step, len(node.Content))
	for idx, node := range node.Content {
		if node.Tag != "!!map" {
			return fmt.Errorf("execution is an array of maps, but got an element of type %q instead", node.Tag)
		}
		var step Step
		stepType := StepType(node.Content[0].Value)
		switch stepType {
		case StepTypeCompute:
			step = &Compute{}
		case StepTypeCall:
			step = &Call{}
		case StepTypeParallel:
			step = &Parallel{}
		case StepTypeLoop:
			step = &Loop{}
		default:
			return fmt.Errorf("unrecognized step type %q in line %d", stepType, node.Line)
		}

		if len(node.Content) > 2 && node.Content[1].Tag == "!!null" {
			// node has both step type and content as a single node, where the step type
			// is actually a key with null value. Example:
			//   execution:
			//   - call:
			//     http: GET something 200
			//     compute: 10ms
			if err := node.Decode(step); err != nil {
				return err
			}
		} else {
			// node has nesting, with type being parent of the content. Example:
			//   execution:
			//   - call:
			//       http: GET something 200
			//       compute: 10ms
			if err := node.Content[1].Decode(step); err != nil {
				return err
			}
		}
		steps[idx] = step
	}
	*e = steps
	return nil
}

func (e Execution) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.toMarshallable())
}

func (e *Execution) UnmarshalJSON(data []byte) error {
	rawSteps := []map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &rawSteps); err != nil {
		return err
	}
	execution := make(Execution, len(rawSteps))
	for idx, rawStep := range rawSteps {
		for stepType, content := range rawStep {
			var step Step
			stepType := StepType(stepType)
			switch stepType {
			case StepTypeCompute:
				step = &Compute{}
			case StepTypeCall:
				step = &Call{}
			case StepTypeParallel:
				step = &Parallel{}
			case StepTypeLoop:
				step = &Loop{}
			default:
				return fmt.Errorf("unrecognized step type %q", stepType)
			}
			if err := json.Unmarshal(content, step); err != nil {
				return err
			}
			execution[idx] = step
		}
	}
	*e = execution
	return nil
}

func (e Execution) toMarshallable() any {
	if len(e) == 0 {
		return nil
	}
	rawExecution := make([]map[string]any, len(e))
	for idx, step := range e {
		rawExecution[idx] = map[string]any{string(step.StepType()): step}
	}
	return rawExecution
}
