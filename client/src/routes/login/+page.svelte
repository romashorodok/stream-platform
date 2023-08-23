<script lang="ts">
	import Input from '$lib/components/base/input.svelte';
	import { useForm } from '$lib/stateful/form';
	import * as yup from 'yup';
	import { login } from '$lib/stores/auth';

	const schema = yup.object({
		username: yup
			.string()
			.required('Username required')
			.matches(/^\S+$/, 'Username should not contain whitespaces')
			.min(3, 'Username requred at least 3 chars'),
		password: yup
			.string()
			.required('Password is required')
			.matches(/[a-zA-Z]/, "Password can contain only Latin latter's")
			.min(8, 'Password required at least 8 chars')
	});

	type Schema = yup.InferType<typeof schema>;

	const { action: FormAction, state, fields } = useForm<Schema>({ schema });

	$: ({ password, username } = state);

	async function OnLogin(e: CustomEvent<Schema>) {
		const payload = e.detail;
		login(payload);
	}

	// export let form: ActionData;
</script>

{$password}
{$username}

<div class="flex h-full">
	<div class="m-auto w-2/4 h-3/5 theme-bg-base theme-fg-base theme-shadow-base">
		<div class="flex flex-col h-full w-full px-8 py-9">
			<h1 class="text-2xl">Login</h1>

			<form use:FormAction on:valid={OnLogin} class="h-full" method="POST">
				<Input label="Username" {...fields.username} />
				<Input label="Password" {...fields.password} />

				<button type="submit">Submit</button>
			</form>
		</div>
	</div>
</div>
