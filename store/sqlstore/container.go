package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	mathRand "math/rand/v2"

	"github.com/google/uuid"
	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/random"

	"zaelix/store"
	"zaelix/store/sqlstore/upgrades"

	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/util/keys"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Container struct {
	db     *dbutil.Database
	log    waLog.Logger
	LIDMap *CachedLIDMap
}

var _ store.DeviceContainer = (*Container)(nil)

func New(ctx context.Context, dialect, address string, log waLog.Logger) (*Container, error) {
	db, err := sql.Open(dialect, address)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	container := NewWithDB(db, dialect, log)
	err = container.Upgrade(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade database: %w", err)
	}
	return container, nil
}

func NewWithDB(db *sql.DB, dialect string, log waLog.Logger) *Container {
	wrapped, err := dbutil.NewWithDB(db, dialect)
	if err != nil {
		
		panic(err)
	}
	wrapped.UpgradeTable = upgrades.Table
	wrapped.VersionTable = "version"
	return NewWithWrappedDB(wrapped, log)
}

func NewWithWrappedDB(wrapped *dbutil.Database, log waLog.Logger) *Container {
	if log == nil {
		log = waLog.Noop
	}
	return &Container{
		db:     wrapped,
		log:    log,
		LIDMap: NewCachedLIDMap(wrapped),
	}
}

func (c *Container) Upgrade(ctx context.Context) error {
	if c.db.Dialect == dbutil.SQLite {
		var foreignKeysEnabled bool
		err := c.db.QueryRow(ctx, "PRAGMA foreign_keys").Scan(&foreignKeysEnabled)
		if err != nil {
			return fmt.Errorf("failed to check if foreign keys are enabled: %w", err)
		} else if !foreignKeysEnabled {
			return fmt.Errorf("foreign keys are not enabled")
		}
	}

	return c.db.Upgrade(ctx)
}

const getAllDevicesQuery = `
SELECT jid, lid, registration_id, noise_key, identity_key,
       signed_pre_key, signed_pre_key_id, signed_pre_key_sig,
       adv_key, adv_details, adv_account_sig, adv_account_sig_key, adv_device_sig,
       platform, business_name, push_name, facebook_uuid, lid_migration_ts
FROM device
`

const getDeviceQuery = getAllDevicesQuery + " WHERE jid=$1"

func (c *Container) scanDevice(row dbutil.Scannable) (*store.Device, error) {
	var device store.Device
	device.Log = c.log
	device.SignedPreKey = &keys.PreKey{}
	var noisePriv, identityPriv, preKeyPriv, preKeySig []byte
	var account waAdv.ADVSignedDeviceIdentity
	var fbUUID uuid.NullUUID

	err := row.Scan(
		&device.ID, &device.LID, &device.RegistrationID, &noisePriv, &identityPriv,
		&preKeyPriv, &device.SignedPreKey.KeyID, &preKeySig,
		&device.AdvSecretKey, &account.Details, &account.AccountSignature, &account.AccountSignatureKey, &account.DeviceSignature,
		&device.Platform, &device.BusinessName, &device.PushName, &fbUUID, &device.LIDMigrationTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	} else if len(noisePriv) != 32 || len(identityPriv) != 32 || len(preKeyPriv) != 32 || len(preKeySig) != 64 {
		return nil, ErrInvalidLength
	}

	device.NoiseKey = keys.NewKeyPairFromPrivateKey(*(*[32]byte)(noisePriv))
	device.IdentityKey = keys.NewKeyPairFromPrivateKey(*(*[32]byte)(identityPriv))
	device.SignedPreKey.KeyPair = *keys.NewKeyPairFromPrivateKey(*(*[32]byte)(preKeyPriv))
	device.SignedPreKey.Signature = (*[64]byte)(preKeySig)
	device.Account = &account
	device.FacebookUUID = fbUUID.UUID

	c.initializeDevice(&device)

	return &device, nil
}

func (c *Container) GetAllDevices(ctx context.Context) ([]*store.Device, error) {
	res, err := c.db.Query(ctx, getAllDevicesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	sessions := make([]*store.Device, 0)
	for res.Next() {
		sess, scanErr := c.scanDevice(res)
		if scanErr != nil {
			return sessions, scanErr
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (c *Container) GetFirstDevice(ctx context.Context) (*store.Device, error) {
	devices, err := c.GetAllDevices(ctx)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return c.NewDevice(), nil
	} else {
		return devices[0], nil
	}
}

func (c *Container) GetDevice(ctx context.Context, jid types.JID) (*store.Device, error) {
	sess, err := c.scanDevice(c.db.QueryRow(ctx, getDeviceQuery, jid))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return sess, err
}

const (
	insertDeviceQuery = `
		INSERT INTO device (jid, lid, registration_id, noise_key, identity_key,
									  signed_pre_key, signed_pre_key_id, signed_pre_key_sig,
									  adv_key, adv_details, adv_account_sig, adv_account_sig_key, adv_device_sig,
									  platform, business_name, push_name, facebook_uuid, lid_migration_ts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (jid) DO UPDATE
			SET lid=excluded.lid,
				platform=excluded.platform,
				business_name=excluded.business_name,
				push_name=excluded.push_name,
				lid_migration_ts=excluded.lid_migration_ts
	`
	deleteDeviceQuery = `DELETE FROM device WHERE jid=$1`
)

func (c *Container) NewDevice() *store.Device {
	device := &store.Device{
		Log:       c.log,
		Container: c,

		NoiseKey:       keys.NewKeyPair(),
		IdentityKey:    keys.NewKeyPair(),
		RegistrationID: mathRand.Uint32(),
		AdvSecretKey:   random.Bytes(32),
	}
	device.SignedPreKey = device.IdentityKey.CreateSignedPreKey(1)
	return device
}

var ErrDeviceIDMustBeSet = errors.New("device JID must be known before accessing database")

func (c *Container) Close() error {
	if c != nil && c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *Container) DB() *sql.DB {
	return c.db.RawDB
}

func (c *Container) PutDevice(ctx context.Context, device *store.Device) error {
	if device.ID == nil {
		return ErrDeviceIDMustBeSet
	}
	_, err := c.db.Exec(ctx, insertDeviceQuery,
		device.ID, device.LID, device.RegistrationID, device.NoiseKey.Priv[:], device.IdentityKey.Priv[:],
		device.SignedPreKey.Priv[:], device.SignedPreKey.KeyID, device.SignedPreKey.Signature[:],
		device.AdvSecretKey, device.Account.Details, device.Account.AccountSignature, device.Account.AccountSignatureKey, device.Account.DeviceSignature,
		device.Platform, device.BusinessName, device.PushName, uuid.NullUUID{UUID: device.FacebookUUID, Valid: device.FacebookUUID != uuid.Nil},
		device.LIDMigrationTimestamp,
	)

	if !device.Initialized {
		c.initializeDevice(device)
	}
	return err
}

func (c *Container) initializeDevice(device *store.Device) {
	innerStore := NewSQLStore(c, *device.ID)
	device.Identities = innerStore
	device.Sessions = innerStore
	device.PreKeys = innerStore
	device.SenderKeys = innerStore
	device.AppStateKeys = innerStore
	device.AppState = innerStore
	device.Contacts = innerStore
	device.ChatSettings = innerStore
	device.MsgSecrets = innerStore
	device.PrivacyTokens = innerStore
	device.EventBuffer = innerStore
	device.LIDs = c.LIDMap
	device.Container = c
	device.Initialized = true
}

func (c *Container) DeleteDevice(ctx context.Context, store *store.Device) error {
	if store.ID == nil {
		return ErrDeviceIDMustBeSet
	}
	_, err := c.db.Exec(ctx, deleteDeviceQuery, store.ID)
	return err
}
