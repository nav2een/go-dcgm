package dcgm

/*
#include "dcgm_agent.h"
#include "dcgm_structs.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type GroupHandle struct{ handle C.dcgmGpuGrp_t }

type Group struct {
	Count      uint
	EntityList []GroupEntityPair
	GroupName  string
	Version    uint
}

func CreateGroup(groupName string) (goGroupId GroupHandle, err error) {
	var cGroupId C.dcgmGpuGrp_t
	cname := C.CString(groupName)
	defer freeCString(cname)

	result := C.dcgmGroupCreate(handle.handle, C.DCGM_GROUP_EMPTY, cname, &cGroupId)
	if err = errorString(result); err != nil {
		return goGroupId, fmt.Errorf("error creating group: %s", err)
	}

	goGroupId = GroupHandle{cGroupId}
	return
}

// Create group and return group id as uint64 instead of returning group handle
func CreateGroupWithId(groupName string) (groupId uint64, err error) {
	var cGroupId C.dcgmGpuGrp_t
	cname := C.CString(groupName)
	defer freeCString(cname)

	result := C.dcgmGroupCreate(handle.handle, C.DCGM_GROUP_EMPTY, cname, &cGroupId)
	if err = errorString(result); err != nil {
		return groupId, fmt.Errorf("error creating group: %s", err)
	}

	groupId = uint64(cGroupId)
	return
}

func NewDefaultGroup(groupName string) (GroupHandle, error) {
	var cGroupId C.dcgmGpuGrp_t

	cname := C.CString(groupName)
	defer freeCString(cname)

	result := C.dcgmGroupCreate(handle.handle, C.DCGM_GROUP_DEFAULT, cname, &cGroupId)
	if err := errorString(result); err != nil {
		return GroupHandle{}, fmt.Errorf("error creating group: %s", err)
	}

	return GroupHandle{cGroupId}, nil
}

func AddToGroup(groupId GroupHandle, gpuId uint) (err error) {
	result := C.dcgmGroupAddDevice(handle.handle, groupId.handle, C.uint(gpuId))
	if err = errorString(result); err != nil {
		return fmt.Errorf("error adding GPU %v to group: %s", gpuId, err)
	}

	return
}

// Add device to group identified by its id instead of group handle.
func GroupAddDevice(groupId uint64, gpuId uint) (err error) {
	result := C.dcgmGroupAddDevice(handle.handle, C.ulong(groupId), C.uint(gpuId))
	if err = errorString(result); err != nil {
		return fmt.Errorf("error adding GPU %v to group: %s", gpuId, err)
	}
	return
}

func AddEntityToGroup(groupId GroupHandle, entityGroupId Field_Entity_Group, entityId uint) (err error) {
	result := C.dcgmGroupAddEntity(handle.handle, groupId.handle, C.dcgm_field_entity_group_t(entityGroupId), C.uint(entityId))
	if err = errorString(result); err != nil {
		return fmt.Errorf("error adding entity group type %v, entity %v to group: %s", entityGroupId, entityId, err)
	}

	return
}

func DestroyGroup(groupId GroupHandle) (err error) {
	result := C.dcgmGroupDestroy(handle.handle, groupId.handle)
	if err = errorString(result); err != nil {
		return fmt.Errorf("error destroying group: %s", err)
	}

	return
}

// Destroy group with given id. Alternate to passing group handle
func DestroyGroupById(groupId uint64) (err error) {
	result := C.dcgmGroupDestroy(handle.handle, C.ulong(groupId))
	if err = errorString(result); err != nil {
		return fmt.Errorf("error destroying group: %s", err)
	}

	return
}

func GroupGetAllIds() (groups []uint64, err error) {
	var groupIdList [C.DCGM_MAX_NUM_GROUPS]C.ulong
	var count C.uint

	result := C.dcgmGroupGetAllIds(handle.handle, &groupIdList[0], &count)
	if err := errorString(result); err != nil {
		return groups, fmt.Errorf("error getting groups count: %s", err)
	}
	numGroups := int(count)
	groups = make([]uint64, numGroups)
	for i := 0; i < numGroups; i++ {
		groups[i] = uint64(groupIdList[i])
	}
	return
}

func GroupGetInfo(groupId uint64) (groupInfo Group, err error) {
	var group C.dcgmGroupInfo_t
	group.version = makeVersion2(unsafe.Sizeof(group))
	result := C.dcgmGroupGetInfo(handle.handle, C.dcgmGpuGrp_t(groupId), &group)
	if err = errorString(result); err != nil {
		return groupInfo, fmt.Errorf("error getting group information: %s", err)
	}
	entityList := make([]GroupEntityPair, len(group.entityList))
	for _, entity := range group.entityList {
		entityList = append(entityList, GroupEntityPair{EntityGroupId: Field_Entity_Group(entity.entityGroupId), EntityId: uint(entity.entityId)})
	}
	groupName := *stringPtr(&group.groupName[0])
	groupInfo = Group{
		Count:      uint(group.count),
		EntityList: entityList,
		GroupName:  groupName,
		Version:    uint(group.version),
	}
	return
}
