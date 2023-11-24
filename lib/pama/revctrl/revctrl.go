package revctrl

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/log"
)

var ErrUnsupported = errors.New("unsupported")

type factoryFunc func(string) models.RevisionController

var controllers = map[string]factoryFunc{}

func register(controllerID string, fn factoryFunc) {
	controllers[controllerID] = fn
}

func New(controllerID string, path string) (models.RevisionController, error) {
	factoryFunc, ok := controllers[controllerID]
	if !ok {
		return nil, errors.New("cannot create revision control instance")
	}
	return factoryFunc(path), nil
}

type detector interface {
	Support() bool
	Root() (string, error)
}

func Detect(path string) (string, string, error) {
	for controllerID, factoryFunc := range controllers {
		rc, ok := factoryFunc(path).(detector)
		if ok && rc.Support() {
			log.Tracef("support found for %v", controllerID)
			root, err := rc.Root()
			if err != nil {
				continue
			}
			log.Tracef("root found in %s", root)
			return controllerID, root, nil
		}
	}
	return "", "", fmt.Errorf("no supported repository found in %s", path)
}
