using Http;

namespace CallCalendar;

@tag("Admin call types")
@route("/v1/admin/call-types")
interface AdminCallTypesOperations {
  /** Lists all non-deleted call types in the administrative order. */
  @get
  listCallTypes(): AdminCallTypeListOkResponse | InternalErrorResponse;

  /** Creates an active call type with a stable public slug. */
  @post
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
  listSchedules(): AdminScheduleListOkResponse | InternalErrorResponse;

  /** Creates a named schedule with all seven ISO weekdays supplied explicitly. */
  @post
  createSchedule(@body body: CreateScheduleRequest):
    | AdminScheduleCreatedResponse
    | InvalidRequestResponse
    | ScheduleNameConflictResponse
    | UnsupportedMediaTypeResponse
    | ValidationErrorResponse
    | InternalErrorResponse;

  /** Replaces a schedule name and all seven days of weekly availability. */
  @put
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
  getWorkingHours(): WorkingHoursOkResponse | InternalErrorResponse;

  /** Replaces all seven days of global working hours. */
  @put
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
