package state

import (
	"os"
	"path/filepath"

	"github.com/f1shl3gs/gossiping/tasks/targetpb"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type Generator struct {
	path   string
	logger *zap.Logger
}

func New(path string, logger *zap.Logger) *Generator {
	return &Generator{
		path:   path,
		logger: logger,
	}
}

func (gen *Generator) OnUpdate(me *targetpb.MeshEntry) {
	if me.Status == targetpb.Status_Active {
		gen.update(me)
	} else {
		gen.backup(me)
	}
}

// we do not delete state file immediately,
// rename it to XXXX.yml.bak, so we restore it from state file
func (gen *Generator) backup(me *targetpb.MeshEntry) {
	fn := gen.filename(me)
	err := os.Rename(fn, fn+".bak")
	if err != nil {
		gen.logger.Warn("state file saved failed",
			zap.String("fn", fn),
			zap.Error(err))
	}
}

func (gen *Generator) update(me *targetpb.MeshEntry) {
	fn := gen.filename(me)
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0644)
	if err != nil {
		gen.logger.Warn("open state file failed",
			zap.String("fn", fn),
			zap.Error(err))
	}

	defer f.Close()

	err = yaml.NewEncoder(f).Encode(me.Targetgroup)
	if err != nil {
		gen.logger.Warn("write state file failed",
			zap.String("fn", fn),
			zap.Error(err))
	}
}

func (gen *Generator) gc() {

}

func (gen *Generator) filename(me *targetpb.MeshEntry) string {
	return filepath.Join(gen.path, me.Name+".yml")
}
