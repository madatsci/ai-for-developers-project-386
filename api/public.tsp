using Http;

namespace CallCalendar;

@tag("Public call types")
@route("/v1/public/call-types")
interface PublicCallTypesOperations {
  /** Lists active call types ordered by duration and then name. */
  @get
  @TypeSpec.OpenAPI.operationId("listPublicCallTypes")
  listCallTypes(): PublicCallTypeListOkResponse | InternalErrorResponse;

  /** Reads an active or disabled call type by its stable public slug. */
  @get
  @TypeSpec.OpenAPI.operationId("getPublicCallType")
  @route("/{slug}")
  getCallType(@path slug: CallTypeSlug):
    | PublicCallTypeOkResponse
    | CallTypeNotFoundResponse
    | InternalErrorResponse;

  /**
   * Returns candidate slots for today and the following 13 Moscow dates.
   * Reading availability does not reserve a slot.
   */
  @get
  @TypeSpec.OpenAPI.operationId("getPublicCallTypeAvailability")
  @route("/{slug}/availability")
  getAvailability(@path slug: CallTypeSlug):
    | AvailabilityOkResponse
    | CallTypeNotFoundResponse
    | CallTypeUnavailableResponse
    | InternalErrorResponse;
}

@tag("Public calls")
@route("/v1/public/calls")
interface PublicCallsOperations {
  /**
   * Atomically rechecks availability and creates one immediately confirmed call.
   * Repeating the same normalized request with the same key returns the original 201 response.
   */
  @post
  @TypeSpec.OpenAPI.operationId("createCall")
  createCall(
    @header("Idempotency-Key") idempotency_key: IdempotencyKey,
    @body body: CreateCallRequest,
  ):
    | CreatedCallResponse
    | InvalidRequestResponse
    | CallTypeChangedResponse
    | CallTypeUnavailableResponse
    | SlotUnavailableResponse
    | IdempotencyConflictResponse
    | UnsupportedMediaTypeResponse
    | ValidationErrorResponse
    | InternalErrorResponse;
}
