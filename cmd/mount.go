package cmd

import (
	"archive/tar"
	"context"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/JakWai01/sile-fystem/internal/logging"
	sf "github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/jacobsa/fuse"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/encryption"
	sfs "github.com/pojntfx/stfs/pkg/fs"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/pojntfx/stfs/pkg/signature"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	driveFlag      = "drive"
	mountpointFlag = "mountpoint"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a folder on a given path",
	RunE: func(cmd *cobra.Command, args []string) error {

		tm := tape.NewTapeManager(
			viper.GetString(driveFlag),
			20,
			false,
		)

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		metadataConfig := config.MetadataConfig{
			Metadata: metadataPersister,
		}

		pipeConfig := config.PipeConfig{
			Compression: config.NoneKey,
			Encryption:  config.NoneKey,
			Signature:   config.NoneKey,
			RecordSize:  20,
		}

		cryptoConfig := config.CryptoConfig{
			Recipient: config.NoneKey,
			Identity:  config.NoneKey,
			Password:  config.NoneKey,
		}

		jsonLogger := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		readOps := operations.NewOperations(
			config.BackendConfig{
				GetWriter:   tm.GetWriter,
				CloseWriter: tm.Close,

				GetReader:   tm.GetReader,
				CloseReader: tm.Close,

				GetDrive:   tm.GetDrive,
				CloseDrive: tm.Close,
			},
			metadataConfig,
			pipeConfig,
			cryptoConfig,
			func(event *config.HeaderEvent) {
				jsonLogger.Debug("Header read", event)
			},
		)

		writeOps := operations.NewOperations(
			config.BackendConfig{
				GetWriter:   tm.GetWriter,
				CloseWriter: tm.Close,

				GetReader:   tm.GetReader,
				CloseReader: tm.Close,

				GetDrive:   tm.GetDrive,
				CloseDrive: tm.Close,
			},
			metadataConfig,
			pipeConfig,
			cryptoConfig,
			func(event *config.HeaderEvent) {
				jsonLogger.Debug("Header write", event)
			},
		)

		stfs := sfs.NewSTFS(
			readOps,
			writeOps,

			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			"balanced",
			func() (cache.WriteCache, func() error, error) {
				return cache.NewCacheWrite(
					filepath.Join("/tmp/stfs", "write"),
					"file",
				)
			},
			true, // FTP needs read permission for `STOR` command even if O_WRONLY is set

			func(hdr *config.Header) {
				jsonLogger.Trace("Header transform", hdr)
			},
			jsonLogger,
		)

		root, err := metadataPersister.GetRootPath(context.Background())
		if err != nil {
			if err == config.ErrNoRootDirectory {
				root = "/"

				drive, err := tm.GetDrive()
				if err == nil {
					err = recovery.Index(
						config.DriveReaderConfig{
							Drive:          drive.Drive,
							DriveIsRegular: drive.DriveIsRegular,
						},
						config.DriveConfig{
							Drive:          drive.Drive,
							DriveIsRegular: drive.DriveIsRegular,
						},
						metadataConfig,
						pipeConfig,
						cryptoConfig,

						20,
						0,
						0,
						true,
						0,

						func(hdr *tar.Header, i int) error {
							return encryption.DecryptHeader(hdr, config.NoneKey, config.NoneKey)
						},
						func(hdr *tar.Header, isRegular bool) error {
							return signature.VerifyHeader(hdr, isRegular, config.NoneKey, config.NoneKey)
						},

						func(hdr *config.Header) {
							jsonLogger.Debug("Header read", hdr)
						},
					)
					if err != nil {
						if err := tm.Close(); err != nil {
							return err
						}

						if err := stfs.MkdirRoot(root, os.ModePerm); err != nil {
							return err
						}
					}
				} else if os.IsNotExist(err) {
					if err := tm.Close(); err != nil {
						return err
					}

					if err := stfs.MkdirRoot(root, os.ModePerm); err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				return err
			}
		}

		fs, err := cache.NewCacheFilesystem(
			stfs,
			root,
			config.NoneKey,
			time.Second*3600,
			filepath.Join("/tmp/stfs", "filesystem"),
		)
		if err != nil {
			return err
		}

		serve := sf.NewFileSystem(currentUid(), currentGid(), viper.GetString(mountpointFlag), root, jsonLogger, fs)
		cfg := &fuse.MountConfig{}

		mfs, err := fuse.Mount(viper.GetString(mountpointFlag), serve, cfg)
		if err != nil {
			log.Fatalf("Mount: %v", err)
		}

		if err := mfs.Join(context.Background()); err != nil {
			log.Fatalf("Join %v", err)
		}

		return nil
	},
}

func init() {
	mountCmd.PersistentFlags().String(driveFlag, "", "Tape drive or tar archive to use as backend")
	mountCmd.PersistentFlags().String(mountpointFlag, "", "Mountpoint to use for FUSE")

	// Bind env variables
	if err := viper.BindPFlags(mountCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.SetEnvPrefix("sile-fystem")
	viper.AutomaticEnv()
}

func currentUid() uint32 {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	uid, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(uid)
}

func currentGid() uint32 {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	gid, err := strconv.ParseUint(user.Gid, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(gid)
}
