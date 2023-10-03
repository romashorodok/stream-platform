import type { ActionReturn } from 'svelte/action';
import { writable, type Writable } from 'svelte/store';
import type * as yup from 'yup';
import { ValidationError } from 'yup';

type FormSubmit<F> = (e: CustomEvent<F>) => void;

type FormActionAttributes<F> = { 'on:valid': FormSubmit<F> };

type FormActionParameter = null;

type FormActionReturn<F> = ActionReturn<FormActionParameter, FormActionAttributes<F>>;

type WritableForm<F> = { [key in keyof F]: Writable<F[key]> };

function nullableWritableForm<F>(schema: yup.AnyObjectSchema): WritableForm<F> {
	const keys = Object.keys(schema.fields) as (keyof F)[];
	const nullableForm = {} as WritableForm<F>;

	for (const key of keys) {
		nullableForm[key] = writable(null as F[keyof F]);
	}

	return nullableForm;
}

type FieldActionReturn = ActionReturn<{}, {}>;

type WritableError = Writable<Array<String>>;

const FieldActionFactory = <F, Key>(
	state: Writable<F>,
	name: Key,
	errors: WritableError,
	validator: (value: any) => Promise<any>
) =>
	function (_: HTMLInputElement): FieldActionReturn {
		const unsubscribe = state.subscribe(async (value) => {
			if (value === '' || value === null) return;

			try {
				await validator({ [name as string]: value });

				errors.set([]);
			} catch (e) {
				if (e instanceof ValidationError) {
					const messages = [] as Array<String>;

					e.inner.forEach((error: yup.ValidationError) => {
						messages.push(error.message);
					});

					errors.set(messages);

					return;
				}

				if (e instanceof Error) errors.set([e.message]);
			}
		});

		return {
			destroy: unsubscribe
		};
	};

type WritableFieldsErrors<F> = { [key in keyof F]: WritableError };

type FormFields<F> = {
	[key in keyof F]: {
		name: key;
		action: Function;
		state: WritableForm<F>[keyof F];
		errors: WritableError;
	};
};

function nullableFormFields<F>(
	schema: yup.AnyObjectSchema,
	writableForm: WritableForm<F>,
	fieldsErrors: WritableFieldsErrors<F>
): FormFields<F> {
	const keys = Object.keys(schema.fields) as (keyof F)[];
	const nullableFields = {} as FormFields<F>;

	for (const key of keys) {
		const state = writableForm[key];
		const errors = fieldsErrors[key];
		const validator = (value: any) =>
			schema.validateAt(key as string, value, {
				abortEarly: false
			});
		const action = FieldActionFactory(state, key, errors, validator);

		nullableFields[key] = { name: key, state, action, errors };
	}

	return nullableFields;
}

function nullableFieldsErrors<F>(schema: yup.AnyObjectSchema): WritableFieldsErrors<F> {
	const keys = Object.keys(schema.fields) as (keyof F)[];
	const nullableFields = {} as WritableFieldsErrors<F>;

	for (const key of keys) nullableFields[key] = writable(new Array<String>());

	return nullableFields;
}

function handleFormSubmitError<F>(e: Error, fieldsErrors: WritableFieldsErrors<F>) {
	if (e instanceof ValidationError) {
		const messages = {} as { [key in keyof F]: Array<String> };

		e.inner.forEach((error: yup.ValidationError) => {
			const errorKey = error.path as keyof typeof messages;

			const undefinedErrors = !messages[errorKey];

			if (undefinedErrors) messages[errorKey] = new Array<String>();

			messages[errorKey].push(error.message);
		});

		for (const [key, errors] of Object.entries(messages)) {
			const keyF = key as keyof typeof fieldsErrors;
			fieldsErrors[keyF].set(errors as String[]);
		}
	}
}

type FormResult<F> = {
	action: (node: HTMLFormElement) => FormActionReturn<F>;
	state: WritableForm<F>;
	fields: FormFields<F>;
};

type FormParams = {
	schema: yup.AnyObjectSchema;
};

export function useForm<F>({ schema }: FormParams): FormResult<F> {
	const writableForm = nullableWritableForm<F>(schema);
	const fieldsErrors = nullableFieldsErrors<F>(schema);
	const formFields = nullableFormFields<F>(schema, writableForm, fieldsErrors);

	function formAction<F>(node: HTMLFormElement): FormActionReturn<F> {
		const validateOnSubmit = async (e: Event) => {
			e.preventDefault();
			let result = {};

			for (const [key, value] of Object.entries(writableForm)) {
				const state = value as Writable<keyof F>;

				const unsubscribe = state.subscribe((val) => {
					result = { [key]: val, ...result };
				});
				unsubscribe();
			}

			try {
				const valid = await schema.validate(result, { abortEarly: false });
				node.dispatchEvent(new CustomEvent('valid', { detail: valid }));
			} catch (e) {
				handleFormSubmitError(e as Error, fieldsErrors);
			}
		};

		node.addEventListener('submit', validateOnSubmit);
		node.autocomplete = 'off';

		return {
			destroy() {
				node.removeEventListener('submit', validateOnSubmit);
			}
		};
	}

	return {
		action: formAction,
		state: writableForm,
		fields: formFields
	};
}
