package user

type (
	// Users defines services needed for user creation, storage, update and removal
	Users interface {
		// Create a new user alias@network with the given name and roles. The
		// returned user will have a unique ID which will be uses later in every
		// other function
		Create(network, alias, name string, rolzs Roles) (*User, error)
		// Find searches for a user with an alias 'alias@network' and returns it.
		Find(network, alias string) (*User, error)
		// Update the user values from the user with the given internal UID. Returns
		// the new and updated user strcture
		Update(uid, username string, rolz Roles) (*User, error)
		// Delete the user with the given UID and returns the deleted user struct.
		Delete(uid string) (*User, error)
		// GetAll returns all users in the datastore
		GetAll() ([]User, error)
		// Get returns the user with the given UID
		Get(uid string) (*User, error)
		// AddKey adds a new public key to the keystore of the user with the given
		// UID. Returns a key structure
		AddKey(uid, kid string, pubkey string, fp string) (*Key, error)
		// RemoveKey removes the key with the given id from the keystore of the given
		// user.
		RemoveKey(uid, kid string) (*Key, error)
		// AddAlias adds a new alias 'alias@network' to the list of known aliases
		// of the given users
		AddAlias(uid, network, alias string) (*User, error)
		// RemoveAlias removes the alias from the list of aliases.
		RemoveAlias(uid, network, alias string) (*User, error)
	}
)
