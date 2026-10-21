using Http;

namespace CallCalendar;

@tag("Admin call types")
@route("/v1/admin/call-types")
interface AdminCallTypesOperations {
  /** Lists all non-deleted call types in the administrative order. */
  @get
  @TypeSpec.OpenAPI.operationId("listAdminCallTypes")
  listCallTypes(): AdminCallTypeListOkResponse | InternalErrorResponse;

  /** Creates an active call type with a stable public slug. */
  @post
  @TypeSpec.OpenAPI.operationId("createAdminCallType")
  createCallType(@body body: CreateCallTypeRequest):
    | AdminCallTypeCreatedResponse
    | InvalidRequestResponse
    | CallTypeNameConflictResponse
    | ScheduleUnavailableResponse
    | UnsupportedMediaTypeResponse
    | ValidationErrorResponse
    | InternalErrorResponse;

  /** Replaces the complete editable state of a call type. */
  @put
  @TypeSpec.OpenAPI.operationId("updateAdminCallType")
  @route("/{id}")
  updateCallType(
    @path id: ResourceId,
    @header("If-Match") if_match: EntityTag,
    @body body: UpdateCallTypeRequest,
  ):
    | AdminCallTypeUpdatedResponse
    | InvalidRequestResponse
    | CallTypeNotFoundResponse
    | CallTypeNameConflictResponse
    | ScheduleUnavailableResponse
    | VersionConflictResponse
    | UnsupportedMediaTypeResponse
    | ValidationErrorResponse
    | PreconditionRequiredResponse
    | InternalErrorResponse;

  /** Soft-deletes a call type which has no call history. */
  @delete
  @TypeSpec.OpenAPI.operationId("deleteAdminCallType")
  @route("/{id}")
  deleteCallType(
    @path id: ResourceId,
    @header("If-Match") if_match: EntityTag,
  ):
    | NoContentResponse
    | InvalidRequestResponse
    | CallTypeNotFoundResponse
    | CallTypeHasCallsResponse
    | VersionConflictResponse
    | PreconditionRequiredResponse
    | InternalErrorResponse;
}

@tag("Admin schedules")
@route("/v1/admin/schedules")
interface AdminSchedulesOperations {
  /** Lists all schedules ordered by name. */
  @get
  @TypeSpec.OpenAPI.operationId("listAdminSchedules")
  listSchedules(): AdminScheduleListOkResponse | InternalErrorResponse;

  /** Creates a named schedule with all seven ISO weekdays supplied explicitly. */
  @post
  @TypeSpec.OpenAPI.operationId("createAdminSchedule")
  createSchedule(@body body: CreateScheduleRequest):
    | AdminScheduleCreatedResponse
    | InvalidRequestResponse
    | ScheduleNameConflictResponse
    | UnsupportedMediaTypeResponse
    | ValidationErrorResponse
    | InternalErrorResponse;

  /** Replaces a schedule name and all seven days of weekly availability. */
  @put
  @TypeSpec.OpenAPI.operationId("updateAdminSchedule")
  @route("/{id}")
  updateSchedule(
    @path id: ResourceId,
    @header("If-Match") if_match: EntityTag,
    @body body: UpdateScheduleRequest,
  ):
    | AdminScheduleUpdatedResponse
    | InvalidRequestResponse
    | ScheduleNotFoundResponse
    | ScheduleNameConflictResponse
    | VersionConflictResponse
    | UnsupportedMediaTypeResponse
    | ValidationErrorResponse
    | PreconditionRequiredResponse
    | InternalErrorResponse;

  /** Physically deletes an unused schedule which is not the last schedule. */
  @delete
  @TypeSpec.OpenAPI.operationId("deleteAdminSchedule")
  @route("/{id}")
  deleteSchedule(
    @path id: ResourceId,
    @header("If-Match") if_match: EntityTag,
  ):
    | NoContentResponse
    | InvalidRequestResponse
    | ScheduleNotFoundResponse
    | ScheduleInUseResponse
    | LastScheduleResponse
    | VersionConflictResponse
    | PreconditionRequiredResponse
    | InternalErrorResponse;
}

@tag("Admin working hours")
@route("/v1/admin/working-hours")
interface AdminWorkingHoursOperations {
  /** Reads the single global working-hours resource. */
  @get
  @TypeSpec.OpenAPI.operationId("getAdminWorkingHours")
  getWorkingHours(): WorkingHoursOkResponse | InternalErrorResponse;

  /** Replaces all seven days of global working hours. */
  @put
  @TypeSpec.OpenAPI.operationId("updateAdminWorkingHours")
  updateWorkingHours(
    @header("If-Match") if_match: EntityTag,
    @body body: UpdateWorkingHoursRequest,
  ):
    | WorkingHoursOkResponse
    | InvalidRequestResponse
    | VersionConflictResponse
    | UnsupportedMediaTypeResponse
    | ValidationErrorResponse
    | PreconditionRequiredResponse
    | InternalErrorResponse;
}

@tag("Admin calls")
@route("/v1/admin/calls")
interface AdminCallsOperations {
  /**
   * Lists a stable 50-item snapshot of upcoming or past calls.
   * A cursor is bound to the snapshot, period, and filters which created it.
   */
  @get
  @TypeSpec.OpenAPI.operationId("listAdminCalls")
  listCalls(
    @query period?: CallPeriod = CallPeriod.Upcoming,
    @query call_type_id?: ResourceId,
    @query q?: CallSearchQuery,
    @query cursor?: PageCursor,
  ):
    | AdminCallPageOkResponse
    | InvalidCursorResponse
    | ValidationErrorResponse
    | InternalErrorResponse;
}
