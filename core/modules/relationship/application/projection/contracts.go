package projection

import sharedevents "wechat-clone/core/shared/contracts/events"

const (
	EventRelationshipPairFriendRequestSent      = sharedevents.EventRelationshipPairFriendRequestSent
	EventRelationshipPairFriendRequestCancelled = sharedevents.EventRelationshipPairFriendRequestCancelled
	EventRelationshipPairFriendRequestAccepted  = sharedevents.EventRelationshipPairFriendRequestAccepted
	EventRelationshipPairFriendRequestRejected  = sharedevents.EventRelationshipPairFriendRequestRejected
	EventRelationshipPairFollowed               = sharedevents.EventRelationshipPairFollowed
	EventRelationshipPairUnfollowed             = sharedevents.EventRelationshipPairUnfollowed
	EventRelationshipPairUnfriended             = sharedevents.EventRelationshipPairUnfriended
	EventRelationshipPairBlocked                = sharedevents.EventRelationshipPairBlocked
	EventRelationshipPairUnblocked              = sharedevents.EventRelationshipPairUnblocked
)
