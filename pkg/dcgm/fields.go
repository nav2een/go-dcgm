package dcgm

/*
#include "dcgm_agent.h"
#include "dcgm_structs.h"

// wrapper for go callback function
extern int listFieldValues_cgo(dcgm_field_entity_group_t entityGroupId, dcgm_field_eid_t entityId, dcgmFieldValue_v1 *values, int numValues, void *userData);
*/
import "C"
import (
	"fmt"
	"runtime/cgo"
	"unicode"
	"unsafe"
)

const (
	defaultUpdateFreq     = 30000000 // usec
	defaultMaxKeepAge     = 0        // sec
	defaultMaxKeepSamples = 1        // Keep one sample by default since we only ask for latest
)

type FieldMeta struct {
	FieldId     Short
	FieldType   byte
	Size        byte
	Tag         string
	Scope       int
	NvmlFieldId int
	EntityLevel Field_Entity_Group
}

type FieldHandle struct{ handle C.dcgmFieldGrp_t }

type FieldGroup struct {
	Id          uint64
	Name        string
	FieldIds    []uint
	NumFieldIds uint
	Version     uint
}

type FieldGroups struct {
	Version        uint
	NumFieldGroups uint
	FieldGroups    []FieldGroup
}

func FieldGroupCreate(fieldsGroupName string, fields []Short) (fieldsId FieldHandle, err error) {
	var fieldsGroup C.dcgmFieldGrp_t
	cfields := *(*[]C.ushort)(unsafe.Pointer(&fields))

	groupName := C.CString(fieldsGroupName)
	defer freeCString(groupName)

	result := C.dcgmFieldGroupCreate(handle.handle, C.int(len(fields)), &cfields[0], groupName, &fieldsGroup)
	if err = errorString(result); err != nil {
		return fieldsId, fmt.Errorf("error creating DCGM fields group: %s", err)
	}

	fieldsId = FieldHandle{fieldsGroup}
	return
}

// Create field group and return its id instead of field group handle
func FieldGroupCreateAndReturnId(fieldsGroupName string, fields []Short) (fieldsId uint64, err error) {
	var fieldsGroup C.dcgmFieldGrp_t
	cfields := *(*[]C.ushort)(unsafe.Pointer(&fields))

	groupName := C.CString(fieldsGroupName)
	defer freeCString(groupName)

	result := C.dcgmFieldGroupCreate(handle.handle, C.int(len(fields)), &cfields[0], groupName, &fieldsGroup)
	if err = errorString(result); err != nil {
		return fieldsId, fmt.Errorf("error creating DCGM fields group: %s", err)
	}

	fieldsId = uint64(fieldsGroup)
	return
}

func FieldGroupDestroy(fieldsGroup FieldHandle) (err error) {
	result := C.dcgmFieldGroupDestroy(handle.handle, fieldsGroup.handle)
	if err = errorString(result); err != nil {
		return fmt.Errorf("error destroying DCGM fields group: %s", err)
	}
	return
}

// Destroy field group for given ID
func FieldGroupDestroyById(fieldGroupId uint64) (err error) {
	result := C.dcgmFieldGroupDestroy(handle.handle, C.ulong(fieldGroupId))
	if err = errorString(result); err != nil {
		return fmt.Errorf("error destroying DCGM fields group: %s", err)
	}
	return
}

func FieldGroupGetAll() (fieldGroups FieldGroups, err error) {
	var allGroupInfo C.dcgmAllFieldGroup_t
	allGroupInfo.version = makeVersion1(unsafe.Sizeof(allGroupInfo))
	result := C.dcgmFieldGroupGetAll(handle.handle, &allGroupInfo)
	if err = errorString(result); err != nil {
		return fieldGroups, fmt.Errorf("error getting field groups: %s", err)
	}
	fieldGroups = FieldGroups{
		Version:        uint(allGroupInfo.version),
		NumFieldGroups: uint(allGroupInfo.numFieldGroups),
		FieldGroups:    make([]FieldGroup, uint(allGroupInfo.numFieldGroups)),
	}
	for _, fieldGroup := range allGroupInfo.fieldGroups {
		numFields := int(fieldGroup.numFieldIds)
		fieldIds := make([]uint, numFields)
		for i := 0; i < numFields; i++ {
			fieldIds[i] = uint(fieldGroup.fieldIds[i])
		}
		fieldGroups.FieldGroups = append(fieldGroups.FieldGroups, FieldGroup{
			Id:       uint64(fieldGroup.fieldGroupId),
			Name:     *stringPtr(&fieldGroup.fieldGroupName[0]),
			FieldIds: fieldIds,
		})
	}
	return
}

func WatchFields(gpuId uint, fieldsGroup FieldHandle, groupName string) (groupId GroupHandle, err error) {
	group, err := CreateGroup(groupName)
	if err != nil {
		return
	}

	err = AddToGroup(group, gpuId)
	if err != nil {
		return
	}

	result := C.dcgmWatchFields(handle.handle, group.handle, fieldsGroup.handle, C.longlong(defaultUpdateFreq), C.double(defaultMaxKeepAge), C.int(defaultMaxKeepSamples))
	if err = errorString(result); err != nil {
		return groupId, fmt.Errorf("error watching fields: %s", err)
	}

	_ = UpdateAllFields()
	return group, nil
}

// Watch fields for given group and field group id
func WatchFields2(groupId uint64, fieldGroupId uint64, updateFreq int64, maxKeepAge float64, maxKeepSamples int32) (err error) {
	result := C.dcgmWatchFields(handle.handle, C.ulong(groupId), C.ulong(fieldGroupId), C.longlong(updateFreq), C.double(maxKeepAge), C.int(maxKeepSamples))
	if err := errorString(result); err != nil {
		return fmt.Errorf("error watching fields: %s", err)
	}
	if err := UpdateAllFields(); err != nil {
		return err
	}
	return
}

func UnwatchFields(groupId uint64, fieldGroupId uint64) (err error) {
	result := C.dcgmUnwatchFields(handle.handle, C.ulong(groupId), C.ulong(fieldGroupId))
	if err := errorString(result); err != nil {
		return fmt.Errorf("error unwatching fields: %s", err)
	}
	return
}

func WatchFieldsWithGroupEx(fieldsGroup FieldHandle, group GroupHandle, updateFreq int64, maxKeepAge float64, maxKeepSamples int32) error {
	result := C.dcgmWatchFields(handle.handle, group.handle, fieldsGroup.handle,
		C.longlong(updateFreq), C.double(maxKeepAge), C.int(maxKeepSamples))

	if err := errorString(result); err != nil {
		return fmt.Errorf("error watching fields: %s", err)
	}

	if err := UpdateAllFields(); err != nil {
		return err
	}

	return nil
}

func WatchFieldsWithGroup(fieldsGroup FieldHandle, group GroupHandle) error {
	return WatchFieldsWithGroupEx(fieldsGroup, group, defaultUpdateFreq, defaultMaxKeepAge, defaultMaxKeepSamples)
}

func GetLatestValuesForFields(gpu uint, fields []Short) ([]FieldValue_v1, error) {
	values := make([]C.dcgmFieldValue_v1, len(fields))
	cfields := *(*[]C.ushort)(unsafe.Pointer(&fields))

	result := C.dcgmGetLatestValuesForFields(handle.handle, C.int(gpu), &cfields[0], C.uint(len(fields)), &values[0])
	if err := errorString(result); err != nil {
		return nil, fmt.Errorf("error watching fields: %s", err)
	}

	return toFieldValue(values), nil
}

func EntityGetLatestValues(entityGroup Field_Entity_Group, entityId uint, fields []Short) ([]FieldValue_v1, error) {
	values := make([]C.dcgmFieldValue_v1, len(fields))
	cfields := (*C.ushort)(unsafe.Pointer(&fields[0]))

	result := C.dcgmEntityGetLatestValues(handle.handle, C.dcgm_field_entity_group_t(entityGroup), C.int(entityId), cfields, C.uint(len(fields)), &values[0])
	if err := errorString(result); err != nil {
		return nil, fmt.Errorf("error getting the latest value for fields: %s", err)
	}

	return toFieldValue(values), nil
}

func EntitiesGetLatestValues(entities []GroupEntityPair, fields []Short, flags uint) ([]FieldValue_v2, error) {
	values := make([]C.dcgmFieldValue_v2, len(fields)*len(entities))
	cfields := (*C.ushort)(unsafe.Pointer(&fields[0]))
	cEntities := make([]C.dcgmGroupEntityPair_t, len(entities))
	cPtrEntities := *(*[]C.dcgmGroupEntityPair_t)(unsafe.Pointer(&cEntities))
	for i, entity := range entities {
		cEntities[i] = C.dcgmGroupEntityPair_t{C.dcgm_field_entity_group_t(entity.EntityGroupId), C.dcgm_field_eid_t(entity.EntityId)}
	}

	result := C.dcgmEntitiesGetLatestValues(handle.handle, &cPtrEntities[0], C.uint(len(entities)), cfields, C.uint(len(fields)), C.uint(flags), &values[0])
	if err := errorString(result); err != nil {
		return nil, fmt.Errorf("error getting the latest value for fields: %s", err)
	}

	return toFieldValue_v2(values), nil
}

// Get latest values for fields using function callback to c
func GetLatestValues(groupId uint64, fieldGroupId uint64) (fieldValues map[GroupEntityPair][]FieldValue_v1, err error) {
	fieldValues = make(map[GroupEntityPair][]FieldValue_v1)
	h := cgo.NewHandle(fieldValues)
	defer h.Delete()
	result := C.dcgmGetLatestValues_v2(handle.handle, C.ulong(groupId), C.ulong(fieldGroupId), (C.dcgmFieldValueEntityEnumeration_f)(unsafe.Pointer(C.listFieldValues_cgo)), unsafe.Pointer(&h))
	if err := errorString(result); err != nil {
		return fieldValues, fmt.Errorf("error getting the latest value for fields: %s", err)
	}
	return
}

//export listFieldValues
func listFieldValues(entityGroupId C.dcgm_field_entity_group_t, entityId C.dcgm_field_eid_t, values *C.dcgmFieldValue_v1, numValues int, userData unsafe.Pointer) int {
	var entityKey GroupEntityPair
	h := *(*cgo.Handle)(userData)
	val := h.Value().(map[GroupEntityPair][]FieldValue_v1)
	entityKey.EntityGroupId = Field_Entity_Group(entityGroupId)
	entityKey.EntityId = uint(entityId)
	fieldValue := FieldValue_v1{
		Version:   uint(values.version),
		FieldId:   uint(values.fieldId),
		FieldType: uint(values.fieldType),
		Status:    int(values.status),
		Ts:        int64(values.ts),
		Value:     values.value,
	}
	val[entityKey] = append(val[entityKey], fieldValue)
	return 0
}

func UpdateAllFields() error {
	waitForUpdate := C.int(1)
	result := C.dcgmUpdateAllFields(handle.handle, waitForUpdate)

	return errorString(result)
}

func toFieldValue(cfields []C.dcgmFieldValue_v1) []FieldValue_v1 {
	fields := make([]FieldValue_v1, len(cfields))
	for i, f := range cfields {
		fields[i] = FieldValue_v1{
			Version:   uint(f.version),
			FieldId:   uint(f.fieldId),
			FieldType: uint(f.fieldType),
			Status:    int(f.status),
			Ts:        int64(f.ts),
			Value:     f.value,
		}
	}

	return fields
}

func (fv FieldValue_v1) Int64() int64 {
	return *(*int64)(unsafe.Pointer(&fv.Value[0]))
}

func (fv FieldValue_v1) Float64() float64 {
	return *(*float64)(unsafe.Pointer(&fv.Value[0]))
}

func (fv FieldValue_v1) String() string {
	return *(*string)(unsafe.Pointer(&fv.Value[0]))
}

func (fv FieldValue_v1) Blob() [4096]byte {
	return fv.Value
}

func toFieldValue_v2(cfields []C.dcgmFieldValue_v2) []FieldValue_v2 {
	fields := make([]FieldValue_v2, len(cfields))
	for i, f := range cfields {
		if uint(f.fieldType) == DCGM_FT_STRING {
			fields[i] = FieldValue_v2{
				Version:       uint(f.version),
				EntityGroupId: Field_Entity_Group(f.entityGroupId),
				EntityId:      uint(f.entityId),
				FieldId:       uint(f.fieldId),
				FieldType:     uint(f.fieldType),
				Status:        int(f.status),
				Ts:            int64(f.ts),
				Value:         f.value,
				StringValue:   stringPtr((*C.char)(unsafe.Pointer(&f.value[0]))),
			}
		} else {
			fields[i] = FieldValue_v2{
				Version:       uint(f.version),
				EntityGroupId: Field_Entity_Group(f.entityGroupId),
				EntityId:      uint(f.entityId),
				FieldId:       uint(f.fieldId),
				FieldType:     uint(f.fieldType),
				Status:        int(f.status),
				Ts:            int64(f.ts),
				Value:         f.value,
				StringValue:   nil,
			}
		}
	}

	return fields
}

func Fv2_Int64(fv FieldValue_v2) int64 {
	return *(*int64)(unsafe.Pointer(&fv.Value[0]))
}

func Fv2_Float64(fv FieldValue_v2) float64 {
	return *(*float64)(unsafe.Pointer(&fv.Value[0]))
}

func FindFirstNonAsciiIndex(value [4096]byte) int {
	for i := 0; i < 4096; i++ {
		if value[i] > unicode.MaxASCII || value[i] < 33 {
			return i
		}
	}

	return 4096
}

func Fv2_String(fv FieldValue_v2) string {
	if fv.FieldType == DCGM_FT_STRING {
		return *fv.StringValue
	} else {
		return string(fv.Value[:])
	}
}

func Fv2_Blob(fv FieldValue_v2) [4096]byte {
	return fv.Value
}

func ToFieldMeta(fieldInfo C.dcgm_field_meta_p) FieldMeta {
	return FieldMeta{
		FieldId:     Short(fieldInfo.fieldId),
		FieldType:   byte(fieldInfo.fieldType),
		Size:        byte(fieldInfo.size),
		Tag:         *stringPtr((*C.char)(unsafe.Pointer(&fieldInfo.tag[0]))),
		Scope:       int(fieldInfo.scope),
		NvmlFieldId: int(fieldInfo.nvmlFieldId),
		EntityLevel: Field_Entity_Group(fieldInfo.entityLevel),
	}
}

func FieldGetById(fieldId Short) FieldMeta {
	return ToFieldMeta(C.DcgmFieldGetById(C.ushort(fieldId)))
}

func FieldsInit() int {
	return int(C.DcgmFieldsInit())
}

func FieldsTerm() int {
	return int(C.DcgmFieldsTerm())
}
