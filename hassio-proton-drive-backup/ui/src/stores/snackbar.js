import { defineStore } from 'pinia';

export const useSnackbarStore = defineStore('snackbar', {
  state: () => ({
    message: '',
    multiLine: false,
    color: 'primary',
    timeout: 3000,
    visible: false,
  }),
  actions: {
    show(options = {}) {
        this.$reset();
  
        Object.assign(this, options);
  
        this.visible = true;
    },
    hideMessage() {
      this.visible = false;
    },
  },
});