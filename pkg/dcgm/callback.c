#include <dcgm_agent.h>

int violationNotify(void *p)
{
    int ViolationRegistration(void *);
    return ViolationRegistration(p);
}

int listFieldValues_cgo(dcgm_field_entity_group_t entityGroupId, dcgm_field_eid_t entityId, dcgmFieldValue_v1 *values, int numValues, void *userData)
{
    int listFieldValues(dcgm_field_entity_group_t entityGroupId, dcgm_field_eid_t entityId, dcgmFieldValue_v1 *values, int numValues, void *userData);
    return listFieldValues(entityGroupId, entityId, values, numValues, userData);
}
