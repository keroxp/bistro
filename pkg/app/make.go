package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/loilo-inc/bistro/pkg/bistro"
	"github.com/loilo-inc/bistro/pkg/docker"
	"github.com/urfave/cli/v3"
)

type makeInput struct {
	imageId    string
	config     string
	dockerHost string
	platform   string
}

func (c *app) makeCommand() *cli.Command {
	var config string
	var dockerHost string
	var imageId string
	var platform string
	return &cli.Command{
		Name: "make",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "tag",
				Aliases:     []string{"t"},
				Usage:       "Image identifier (format: \"[registry/]repository[:tag]\")",
				Destination: &imageId,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "platform",
				Usage:       "Target platform for the image",
				Destination: &platform,
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "Path to the configuration file",
				Value:       "bistro.yml",
				Destination: &config,
			},
			&cli.StringFlag{
				Name:        "docker-host",
				Usage:       "Docker host to connect to",
				Destination: &dockerHost,
				Value:       docker.DefaultSocketPath,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dockerClient := docker.NewClient(dockerHost)
			return c.make(ctx, dockerClient, makeInput{
				imageId:    imageId,
				config:     config,
				dockerHost: dockerHost,
				platform:   platform,
			})
		},
	}
}

func parseImageRef(imageId string) (string, string) {
	parts := strings.Split(imageId, "/")
	if len(parts) == 1 {
		return parseImageTag(parts[0])
	}
	last := parts[len(parts)-1]
	repo, tag := parseImageTag(last)
	return strings.Join(parts[:len(parts)-1], "/") + "/" + repo, tag
}

func parseImageTag(lastComponent string) (string, string) {
	parts := strings.SplitN(lastComponent, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "latest"
}

func (c *app) make(ctx context.Context, dockerClient docker.Client, input makeInput) error {
	recipe, err := bistro.ReadRecipe(input.config)
	if err != nil {
		return err
	}
	timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	imageOut, err := os.CreateTemp("", "image-out")
	if err != nil {
		return err
	}
	defer func() {
		imageOut.Close()
		os.Remove(imageOut.Name())
	}()
	c.printf("Saving base image to temporary file...\n")
	err = dockerClient.Save(timeout, recipe.Build.From, imageOut)
	if err != nil {
		return err
	}
	c.printf("Bundling image...\n")
	fsHandle, err := openFs(fmt.Sprintf("oci:%s", imageOut.Name()))
	if err != nil {
		return err
	}
	defer fsHandle.Close()
	fss := fsHandle.FS()
	di := c.depsFromFs(fss)
	if err := load(di); err != nil {
		return err
	}
	maker := bistro.NewMaker(di, dockerClient)
	repo, tag := parseImageRef(input.imageId)
	made, err := maker.Make(recipe, bistro.MakeInput{
		Repository: repo,
		Tag:        tag,
		Platform:   input.platform,
	})
	if err != nil {
		return err
	}
	c.printf("Image created: %s\n", made)
	return nil
}
