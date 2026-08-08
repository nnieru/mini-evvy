<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AuthLayout from '@/app/layouts/AuthLayout.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { useRegisterMutation } from '../queries/auth'

const form = reactive({
  name: '',
  email: '',
  password: '',
  confirm_password: '',
})
const error = ref('')

const registerMutation = useRegisterMutation()

async function onSubmit() {
  error.value = ''
  if (form.password !== form.confirm_password) {
    error.value = 'Passwords do not match'
    return
  }
  try {
    await registerMutation.mutateAsync(form)
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <AuthLayout title="Create account" description="Start organizing events with mini-evvy.">
    <form class="space-y-4" @submit.prevent="onSubmit">
      <Alert v-if="error" tone="error" :title="error" />
      <Input v-model="form.name" label="Name" required autocomplete="name" />
      <Input v-model="form.email" label="Email" type="email" required autocomplete="email" />
      <Input
        v-model="form.password"
        label="Password"
        type="password"
        required
        autocomplete="new-password"
      />
      <Input
        v-model="form.confirm_password"
        label="Confirm password"
        type="password"
        required
        autocomplete="new-password"
      />
      <Button type="submit" class="w-full" :loading="registerMutation.isPending.value">
        Create account
      </Button>
    </form>
    <p class="mt-6 text-center text-sm text-ink-muted">
      Already have an account?
      <RouterLink class="font-medium text-accent hover:underline" to="/login">
        Sign in
      </RouterLink>
    </p>
  </AuthLayout>
</template>
