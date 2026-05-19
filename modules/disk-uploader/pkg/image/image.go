package image

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.uber.org/zap"

	"github.com/kubevirt/kubevirt-tekton-tasks/modules/shared/pkg/log"

	tar "kubevirt.io/containerdisks/pkg/build"
)

var labelEnvPairs = map[string]string{
	"instancetype.kubevirt.io/default-instancetype":      "INSTANCETYPE_KUBEVIRT_IO_DEFAULT_INSTANCETYPE",
	"instancetype.kubevirt.io/default-instancetype-kind": "INSTANCETYPE_KUBEVIRT_IO_DEFAULT_INSTANCETYPE_KIND",
	"instancetype.kubevirt.io/default-preference":        "INSTANCETYPE_KUBEVIRT_IO_DEFAULT_PREFERENCE",
	"instancetype.kubevirt.io/default-preference-kind":   "INSTANCETYPE_KUBEVIRT_IO_DEFAULT_PREFERENCE_KIND",
}

func DefaultConfig(labels map[string]string) v1.Config {
	var env []string
	for label, envVar := range labelEnvPairs {
		if value, exists := labels[label]; exists {
			env = append(env, fmt.Sprintf("%s=%s", envVar, value))
		}
	}
	return v1.Config{Env: env}
}

func Build(diskPath string, config v1.Config) (v1.Image, error) {
	layer, err := tarball.LayerFromOpener(tar.StreamLayerOpener(diskPath))
	if err != nil {
		return nil, fmt.Errorf("error creating layer from file: %v", err)
	}

	image, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		return nil, fmt.Errorf("error appending layer: %v", err)
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("error getting the image config file: %v", err)
	}
	configFile.Config = config

	image, err = mutate.ConfigFile(image, configFile)
	if err != nil {
		return nil, fmt.Errorf("error setting the image config file: %v", err)
	}
	return image, nil
}

func Push(image v1.Image, imageDestination string, pushTimeout int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(pushTimeout))
	defer cancel()

	ref, err := name.ParseReference(imageDestination)
	if err != nil {
		return fmt.Errorf("error parsing image reference: %v", err)
	}

	// Get total image size for progress tracking by summing all layer sizes
	layers, err := image.Layers()
	if err != nil {
		return fmt.Errorf("error getting image layers: %v", err)
	}

	var totalSize int64
	for _, layer := range layers {
		size, err := layer.Size()
		if err != nil {
			return fmt.Errorf("error getting layer size: %v", err)
		}
		totalSize += size
	}

	log.Logger().Info("Image size calculated", zap.Int64("total_bytes", totalSize), zap.Int("layer_count", len(layers)))

	auth := &authn.Basic{
		Username: os.Getenv("ACCESS_KEY_ID"),
		Password: os.Getenv("SECRET_KEY"),
	}

	progressChan := make(chan v1.Update, 100)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for update := range progressChan {
			if update.Error != nil {
				log.Logger().Error("Upload error", zap.Error(update.Error))
				continue
			}

			if update.Complete > 0 && totalSize > 0 {
				percentage := float64(update.Complete) / float64(totalSize) * 100
				log.Logger().Info("Pushing image progress",
					zap.Float64("percentage", percentage),
					zap.Int64("bytes_uploaded", update.Complete),
					zap.Int64("total_bytes", totalSize))
			} else if update.Complete > 0 {
				log.Logger().Info("Pushing image", zap.Int64("bytes_uploaded", update.Complete))
			}
		}
	}()

	err = remote.Write(ref, image,
		remote.WithAuth(auth),
		remote.WithContext(ctx),
		remote.WithProgress(progressChan),
	)

	// Wait for the progress goroutine to finish processing all updates
	// The library closes progressChan, which will cause the goroutine to exit
	<-done

	if err != nil {
		return fmt.Errorf("error pushing image: %v", err)
	}
	return nil
}
