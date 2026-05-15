package store

import waStore "go.mau.fi/whatsmeow/store"

type IdentityStore = waStore.IdentityStore
type SessionStore = waStore.SessionStore
type PreKeyStore = waStore.PreKeyStore
type SenderKeyStore = waStore.SenderKeyStore
type AppStateSyncKey = waStore.AppStateSyncKey
type AppStateSyncKeyStore = waStore.AppStateSyncKeyStore
type AppStateMutationMAC = waStore.AppStateMutationMAC
type AppStateStore = waStore.AppStateStore
type ContactEntry = waStore.ContactEntry
type RedactedPhoneEntry = waStore.RedactedPhoneEntry
type ContactStore = waStore.ContactStore
type ChatSettingsStore = waStore.ChatSettingsStore
type DeviceContainer = waStore.DeviceContainer
type MessageSecretInsert = waStore.MessageSecretInsert
type MsgSecretStore = waStore.MsgSecretStore
type PrivacyToken = waStore.PrivacyToken
type PrivacyTokenStore = waStore.PrivacyTokenStore
type BufferedEvent = waStore.BufferedEvent
type EventBuffer = waStore.EventBuffer
type LIDMapping = waStore.LIDMapping
type LIDStore = waStore.LIDStore
type AllSessionSpecificStores = waStore.AllSessionSpecificStores
type AllGlobalStores = waStore.AllGlobalStores
type AllStores = waStore.AllStores
type Device = waStore.Device

var MutedForever = waStore.MutedForever
