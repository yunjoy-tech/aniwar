package useractor

import (
	"gitee.com/aniwar2/musae/framework/state"
)

func (u *UserActor) GetStateManager() *UserActor {
	return u
}

func (u *UserActor) Get(stateName string) (*state.KvTable, error) {
	return u.Srv.GetMongoGame(stateName, nil)
}

func (u *UserActor) Set(stateName string, value *state.KvTable) error {
	err := u.Srv.SaveMongoGame(stateName, value, nil)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserActor) SetID(id string) {
	u.ServerImplBase.SetID(id)
	// u.id = id
	u.uid, u.roleId = u.Srv.ConvUAID(id)
}

// func (u *UserActor) ID() string {
//	return u.ID()
// }

// SetAccountId account id for svc invoke
func (u *UserActor) SetUID(uid string) {
	u.uid = uid
}

func (u *UserActor) GetUID() string {
	return u.uid
}

// SetAccountId account id for svc invoke
func (u *UserActor) SetRID(rid uint64) {
	u.roleId = rid
}

func (u *UserActor) GetRID() uint64 {
	return u.roleId
}

func (u *UserActor) Contains(stateName string) (bool, error) {
	_, err := u.Get(stateName)
	// if err != nil || reply == nil || reply.Data == nil || len(reply.Data) == 0 {
	//	return false, err
	// }
	if err != nil {
		return false, err
	}
	return true, nil
}

func (u *UserActor) Save() error {

	return nil
}
