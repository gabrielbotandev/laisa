package app

import (
	"fmt"
	"strings"
)

// ErrNoModel is returned when no local model is available.
type ErrNoModel struct{}

func (ErrNoModel) Error() string {
	return `No model found.

Download one with:
  shai --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini`
}

// ErrModelNotFound is returned when a named model does not exist.
type ErrModelNotFound struct {
	Name string
}

func (e ErrModelNotFound) Error() string {
	models, err := ListModels()
	if err != nil {
		return fmt.Sprintf("Model not found: %s", e.Name)
	}
	return fmt.Sprintf("Model not found: %s\n\nLocal models:\n%s", e.Name, FormatModelList(models))
}

// ErrVenvMissing is returned when the Python venv is not installed.
type ErrVenvMissing struct{}

func (ErrVenvMissing) Error() string {
	return `Python backend is not installed.

Run the installer:
  ./scripts/install.sh`
}

// WrapNPUError wraps NPU failures with a helpful hint.
func WrapNPUError(device string, err error) error {
	if err == nil {
		return nil
	}
	if strings.ToUpper(device) != "NPU" {
		return err
	}
	return fmt.Errorf(`Failed to run model on NPU.

Try CPU:
  shai --device CPU "Say hello"

Original error:
%w`, err)
}
