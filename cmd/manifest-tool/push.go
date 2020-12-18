package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/estesp/manifest-tool/pkg/registry"
	"github.com/estesp/manifest-tool/pkg/types"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

var pushCmd = cli.Command{
	Name:  "push",
	Usage: "push a manifest list/OCI index entry to a registry with provided image details",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "type",
			Value: "docker",
			Usage: "image manifest type: docker (v2.2 manifest list) or oci (v1 index)",
		},
	},
	Subcommands: []cli.Command{
		{
			Name:  "from-spec",
			Usage: "push a manifest list to a registry via a YAML spec",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "ignore-missing",
					Usage: "only warn on missing images defined in YAML spec",
				},
			},
			Action: func(c *cli.Context) {
				filePath := c.Args().First()
				var yamlInput types.YAMLInput

				filename, err := filepath.Abs(filePath)
				if err != nil {
					logrus.Fatalf(fmt.Sprintf("Can't resolve path to %q: %v", filePath, err))
				}
				yamlFile, err := ioutil.ReadFile(filename)
				if err != nil {
					logrus.Fatalf(fmt.Sprintf("Can't read YAML file %q: %v", filePath, err))
				}
				err = yaml.Unmarshal(yamlFile, &yamlInput)
				if err != nil {
					logrus.Fatalf(fmt.Sprintf("Can't unmarshal YAML file %q: %v", filePath, err))
				}

				_, _, err = registry.PushManifestList(c.GlobalString("username"), c.GlobalString("password"), yamlInput, c.Bool("ignore-missing"), c.GlobalBool("insecure"), c.GlobalBool("plain-http"), filepath.Join(c.GlobalString("docker-cfg"), "config.json"))
				if err != nil {
					logrus.Fatal(err)
				}
			},
		},
		{
			Name:  "from-args",
			Usage: "push a manifest list to a registry via CLI arguments",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "platforms",
					Usage: "comma-separated list of the platforms that images should be pushed for",
				},
				cli.StringFlag{
					Name:  "template",
					Usage: "the pattern the source images have. OS and ARCH in that pattern will be replaced with the actual values from the platforms list",
				},
				cli.StringFlag{
					Name:  "target",
					Usage: "the name of the manifest list image that is going to be produced",
				},
				cli.BoolFlag{
					Name:  "ignore-missing",
					Usage: "only warn on missing images defined in platform list",
				},
			},
			Action: func(c *cli.Context) {
				platforms := c.String("platforms")
				templ := c.String("template")
				target := c.String("target")
				srcImages := []types.ManifestEntry{}

				if len(platforms) == 0 || len(templ) == 0 || len(target) == 0 {
					logrus.Fatalf("You must specify all three arguments --platforms, --template and --target")
				}

				platformList := strings.Split(platforms, ",")

				for _, platform := range platformList {
					osArchArr := strings.Split(platform, "/")
					if len(osArchArr) != 2 && len(osArchArr) != 3 {
						logrus.Fatal("The --platforms argument must be a string slice where one value is of the form 'os/arch'")
					}
					variant := ""
					os, arch := osArchArr[0], osArchArr[1]
					if len(osArchArr) == 3 {
						variant = osArchArr[2]
					}
					srcImages = append(srcImages, types.ManifestEntry{
						Image: strings.Replace(strings.Replace(strings.Replace(templ, "ARCH", arch, 1), "OS", os, 1), "VARIANT", variant, 1),
						Platform: ocispec.Platform{
							OS:           os,
							Architecture: arch,
							Variant:      variant,
						},
					})
				}
				yamlInput := types.YAMLInput{
					Image:     target,
					Manifests: srcImages,
				}
				_, _, err := registry.PushManifestList(c.GlobalString("username"), c.GlobalString("password"), yamlInput, c.Bool("ignore-missing"), c.GlobalBool("insecure"), c.GlobalBool("plain-http"), filepath.Join(c.GlobalString("docker-cfg"), "config.json"))
				if err != nil {
					logrus.Fatal(err)
				}
			},
		},
	},
}
