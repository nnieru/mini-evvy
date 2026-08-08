<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AuthLayout from '@/app/layouts/AuthLayout.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { useLoginMutation } from '../queries/auth'

const form = reactive({ email: '', password: '' })
const error = ref('')

const loginMutation = useLoginMutation()

async function onSubmit() {
  error.value = ''
  try {
    await loginMutation.mutateAsync(form)
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <AuthLayout title="Welcome back" description="Sign in to manage your events and seating.">
    <form class="space-y-4" @submit.prevent="onSubmit">
      <Alert v-if="error" tone="error" :title="error" />
      <Input v-model="form.email" label="Email" type="email" required autocomplete="email" />
      <Input
        v-model="form.password"
        label="Password"
        type="password"
        required
        autocomplete="current-password"
      />
      <Button type="submit" class="w-full" :loading="loginMutation.isPending.value">
        Sign in
      </Button>
    </form>
    <p class="mt-6 text-center text-sm text-ink-muted">
      No account?
      <RouterLink class="font-medium text-accent hover:underline" to="/register">
        Create one
      </RouterLink>
    </p>
  </AuthLayout>
</template>
