package cmd

import (
	"archive/tar"
	"context"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/JakWai01/sile-fystem/internal/logging"
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

	sf "github.com/JakWai01/sile-fystem/pkg/filesystem"
)

const (
	mountpointFlag = "mountpoint"

	driveFlag      = "drive"
	recordSizeFlag = "recordSize"
	writeCacheFlag = "writeCache"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a folder on a given path",
	RunE: func(cmd *cobra.Command, args []string) error {

		l := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		tm := tape.NewTapeManager(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			false,
		)

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			panic(err)
		}

		metadataConfig := config.MetadataConfig{
			Metadata: metadataPersister,
		}
		pipeConfig := config.PipeConfig{
			Compression: config.NoneKey,
			Encryption:  config.NoneKey,
			Signature:   config.NoneKey,
			RecordSize:  viper.GetInt(recordSizeFlag),
		}
		backendConfig := config.BackendConfig{
			GetWriter:   tm.GetWriter,
			CloseWriter: tm.Close,

			GetReader:   tm.GetReader,
			CloseReader: tm.Close,

			GetDrive:   tm.GetDrive,
			CloseDrive: tm.Close,
		}
		readCryptoConfig := config.CryptoConfig{}

		readOps := operations.NewOperations(
			backendConfig,
			metadataConfig,
			pipeConfig,
			readCryptoConfig,

			func(event *config.HeaderEvent) {
				l.Debug("Header read", event)
			},
		)
		writeOps := operations.NewOperations(
			backendConfig,
			metadataConfig,

			pipeConfig,
			config.CryptoConfig{},

			func(event *config.HeaderEvent) {
				l.Debug("Header write", event)
			},
		)

		stfs := sfs.NewSTFS(
			readOps,
			writeOps,

			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			config.CompressionLevelFastest,
			func() (cache.WriteCache, func() error, error) {
				return cache.NewCacheWrite(
					viper.GetString(writeCacheFlag),
					config.WriteCacheTypeFile,
				)
			},
			false,

			func(hdr *config.Header) {
				l.Trace("Header transform", hdr)
			},
			l,
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
						readCryptoConfig,

						viper.GetInt(recordSizeFlag),
						0,
						0,
						true,
						0,

						func(hdr *tar.Header, i int) error {
							return encryption.DecryptHeader(hdr, config.NoneKey, nil)
						},
						func(hdr *tar.Header, isRegular bool) error {
							return signature.VerifyHeader(hdr, isRegular, config.NoneKey, nil)
						},

						func(hdr *config.Header) {
							l.Debug("Header read", hdr)
						},
					)
					if err != nil {
						if err := tm.Close(); err != nil {
							panic(err)
						}

						if err := stfs.MkdirRoot(root, os.ModePerm); err != nil {
							panic(err)
						}
					}
				} else if os.IsNotExist(err) {
					if err := tm.Close(); err != nil {
						panic(err)
					}

					if err := stfs.MkdirRoot(root, os.ModePerm); err != nil {
						panic(err)
					}
				} else {
					panic(err)
				}
			} else {
				panic(err)
			}
		}

		fs, err := cache.NewCacheFilesystem(
			stfs,
			root,
			config.NoneKey,
			0,
			"",
		)
		if err != nil {
			panic(err)
		}

		serve := sf.NewFileSystem(currentUid(), currentGid(), viper.GetString(mountpointFlag), root, l, fs)
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
	mountCmd.PersistentFlags().String(mountpointFlag, "/tmp/mount", "Mountpoint to use for FUSE")

	mountCmd.PersistentFlags().String(driveFlag, "/dev/nst0", "Tape drive or tar archive to use as backend")
	mountCmd.PersistentFlags().Int(recordSizeFlag, 20, "Amount of 512-bit blocks per second")
	mountCmd.PersistentFlags().String(writeCacheFlag, filepath.Join(os.TempDir(), "stfs-write-cache"), "Directory to use for write cache")

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
