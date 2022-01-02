package cmd

import (
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
	sfs "github.com/pojntfx/stfs/pkg/fs"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	driveFlag = "drive"
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
			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			config.PipeConfig{
				Compression: "none",
				Encryption:  "none",
				Signature:   "none",
				RecordSize:  20,
			},
			config.CryptoConfig{
				Recipient: "none",
				Identity:  "none",
				Password:  "none",
			},

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
			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			config.PipeConfig{
				Compression: "none",
				Encryption:  "none",
				Signature:   "none",
				RecordSize:  20,
			},
			config.CryptoConfig{
				Recipient: "none",
				Identity:  "none",
				Password:  "none",
			},

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
				// FIXME: Re-index first, and only `Mkdir` if it still fails after indexing, otherwise this would prevent usage of non-indexed, existing tar files

				root = "/"
				if err := stfs.MkdirRoot(root, os.ModePerm); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		fs, err := cache.NewCacheFilesystem(
			stfs,
			root,
			"none",
			time.Second*3600,
			filepath.Join("/tmp/stfs", "filesystem"),
		)
		if err != nil {
			return err
		}

		serve := sf.NewFileSystem(currentUid(), currentGid(), viper.GetString(driveFlag), jsonLogger, fs)
		cfg := &fuse.MountConfig{}

		mfs, err := fuse.Mount(root, serve, cfg)
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
	mountCmd.PersistentFlags().String(driveFlag, "", "Tape drive or tar archive to mount")

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
