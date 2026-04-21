package processor

import (
	"context"
	"reflect"

	relationshipaggregate "wechat-clone/core/modules/relationship/domain/aggregate"
	relationshipprojection "wechat-clone/core/modules/relationship/application/projection"
	"wechat-clone/core/shared/contracts"
	"wechat-clone/core/shared/pkg/stackErr"
)

var projectionEventPayloadTypes = map[string]reflect.Type{
	relationshipprojection.EventRelationshipPairFriendRequestSent:      reflect.TypeOf(relationshipaggregate.EventRelationshipPairFriendRequestSent{}),
	relationshipprojection.EventRelationshipPairFriendRequestCancelled: reflect.TypeOf(relationshipaggregate.EventRelationshipPairFriendRequestCancelled{}),
	relationshipprojection.EventRelationshipPairFriendRequestAccepted:  reflect.TypeOf(relationshipaggregate.EventRelationshipPairFriendRequestAccepted{}),
	relationshipprojection.EventRelationshipPairFriendRequestRejected:  reflect.TypeOf(relationshipaggregate.EventRelationshipPairFriendRequestRejected{}),
	relationshipprojection.EventRelationshipPairFollowed:               reflect.TypeOf(relationshipaggregate.EventRelationshipPairFollowed{}),
	relationshipprojection.EventRelationshipPairUnfollowed:             reflect.TypeOf(relationshipaggregate.EventRelationshipPairUnfollowed{}),
	relationshipprojection.EventRelationshipPairUnfriended:             reflect.TypeOf(relationshipaggregate.EventRelationshipPairUnfriended{}),
	relationshipprojection.EventRelationshipPairBlocked:                reflect.TypeOf(relationshipaggregate.EventRelationshipPairBlocked{}),
	relationshipprojection.EventRelationshipPairUnblocked:              reflect.TypeOf(relationshipaggregate.EventRelationshipPairUnblocked{}),
}

func decodeEventPayload(_ context.Context, eventName string, raw []byte) (any, error) {
	payloadType, ok := projectionEventPayloadTypes[eventName]
	if !ok {
		return nil, nil
	}
	value := reflect.New(payloadType)
	if err := contracts.UnmarshalEventData(raw, value.Interface()); err != nil {
		return nil, stackErr.Error(err)
	}
	return value.Interface(), nil
}
