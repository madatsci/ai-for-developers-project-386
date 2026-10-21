using Http;

namespace CallCalendar;

model NoStoreHeaders {
  @header("Cache-Control")
  cache_control: "no-store";
}

model PublicCallTypeListOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @body body: PublicCallTypeList;
}

model PublicCallTypeOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @body body: PublicCallType;
}

model AvailabilityOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @body body: AvailabilityResponseBody;
}

model CreatedCallResponse extends NoStoreHeaders {
  @statusCode status_code: 201;
  @body body: CreatedCall;
}

model AdminCallTypeListOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @body body: AdminCallTypeList;
}

model AdminCallTypeCreatedResponse extends NoStoreHeaders {
  @statusCode status_code: 201;
  @header("ETag") etag: EntityTag;
  @body body: AdminCallType;
}

model AdminCallTypeUpdatedResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @header("ETag") etag: EntityTag;
  @body body: AdminCallType;
}

model AdminScheduleListOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @body body: AdminScheduleList;
}

model AdminScheduleCreatedResponse extends NoStoreHeaders {
  @statusCode status_code: 201;
  @header("ETag") etag: EntityTag;
  @body body: AdminSchedule;
}

model AdminScheduleUpdatedResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @header("ETag") etag: EntityTag;
  @body body: AdminSchedule;
}

model WorkingHoursOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @header("ETag") etag: EntityTag;
  @body body: WorkingHours;
}

model AdminCallPageOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @body body: AdminCallPage;
}

model NoContentResponse extends NoStoreHeaders {
  @statusCode status_code: 204;
}

model FieldError {
  path: string;
  code: string;
  message: string;
}

@error
model InvalidRequestError {
  code: "invalid_request";
  message: string;
}

@error
model InvalidCursorError {
  code: "invalid_cursor";
  message: string;
}

@error
model CallTypeNotFoundError {
  code: "call_type_not_found";
  message: string;
}

@error
model ScheduleNotFoundError {
  code: "schedule_not_found";
  message: string;
}

@error
model CallTypeNameConflictError {
  code: "call_type_name_conflict";
  message: string;
}

@error
model ScheduleNameConflictError {
  code: "schedule_name_conflict";
  message: string;
}

@error
model CallTypeHasCallsError {
  code: "call_type_has_calls";
  message: string;
}

@error
model ScheduleInUseError {
  code: "schedule_in_use";
  message: string;
}

@error
model LastScheduleError {
  code: "last_schedule";
  message: string;
}

@error
model ScheduleUnavailableError {
  code: "schedule_unavailable";
  message: string;
}

@error
model CallTypeChangedError {
  code: "call_type_changed";
  message: string;
}

@error
model CallTypeUnavailableError {
  code: "call_type_unavailable";
  message: string;
}

@error
model SlotUnavailableError {
  code: "slot_unavailable";
  message: string;
}

@error
model IdempotencyConflictError {
  code: "idempotency_conflict";
  message: string;
}

@error
model VersionConflictError {
  code: "version_conflict";
  message: string;
}

@error
model UnsupportedMediaTypeError {
  code: "unsupported_media_type";
  message: string;
}

@error
model ValidationError {
  code: "validation_error";
  message: string;
  field_errors: FieldError[];
}

@error
model PreconditionRequiredError {
  code: "precondition_required";
  message: string;
}

@error
model InternalError {
  code: "internal_error";
  message: string;
}

model InvalidRequestResponse extends NoStoreHeaders {
  @statusCode status_code: 400;
  @body body: InvalidRequestError;
}

model InvalidCursorResponse extends NoStoreHeaders {
  @statusCode status_code: 400;
  @body body: InvalidCursorError;
}

model CallTypeNotFoundResponse extends NoStoreHeaders {
  @statusCode status_code: 404;
  @body body: CallTypeNotFoundError;
}

model ScheduleNotFoundResponse extends NoStoreHeaders {
  @statusCode status_code: 404;
  @body body: ScheduleNotFoundError;
}

model CallTypeNameConflictResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: CallTypeNameConflictError;
}

model ScheduleNameConflictResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: ScheduleNameConflictError;
}

model CallTypeHasCallsResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: CallTypeHasCallsError;
}

model ScheduleInUseResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: ScheduleInUseError;
}

model LastScheduleResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: LastScheduleError;
}

model ScheduleUnavailableResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: ScheduleUnavailableError;
}

model CallTypeChangedResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: CallTypeChangedError;
}

model CallTypeUnavailableResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: CallTypeUnavailableError;
}

model SlotUnavailableResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: SlotUnavailableError;
}

model IdempotencyConflictResponse extends NoStoreHeaders {
  @statusCode status_code: 409;
  @body body: IdempotencyConflictError;
}

model VersionConflictResponse extends NoStoreHeaders {
  @statusCode status_code: 412;
  @body body: VersionConflictError;
}

model UnsupportedMediaTypeResponse extends NoStoreHeaders {
  @statusCode status_code: 415;
  @body body: UnsupportedMediaTypeError;
}

model ValidationErrorResponse extends NoStoreHeaders {
  @statusCode status_code: 422;
  @body body: ValidationError;
}

model PreconditionRequiredResponse extends NoStoreHeaders {
  @statusCode status_code: 428;
  @body body: PreconditionRequiredError;
}

model InternalErrorResponse extends NoStoreHeaders {
  @statusCode status_code: 500;
  @body body: InternalError;
}

model HealthOkBody {
  status: "ok";
}

model HealthUnavailableBody {
  status: "unavailable";
}

model HealthOkResponse extends NoStoreHeaders {
  @statusCode status_code: 200;
  @body body: HealthOkBody;
}

model HealthUnavailableResponse extends NoStoreHeaders {
  @statusCode status_code: 503;
  @body body: HealthUnavailableBody;
}
