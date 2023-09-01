<template>
  <layout-content :header="$t('commons.button.create')" :back-to="{ name: 'Groups' }">
    <el-row v-loading="loading">
      <el-col :span="4"><br /></el-col>
      <el-col :span="10">
        <div class="grid-content bg-purple-light">
          <el-form ref="form" :model="form" :rules="rules" label-width="150px" label-position="left">
            <el-form-item label="用户组名" prop="name">
              <el-input v-model="form.name" :disabled="this.name"></el-input>
            </el-form-item>
            <el-form-item label="用户组" prop="nickname">
              <el-input v-model="form.nickname"></el-input>
            </el-form-item>
            <el-form-item :label="$t('business.user.role')" prop="roles">
              <el-select v-model="form.roles" multiple filterable style="width: 100%"
                :placeholder="$t('commons.form.select_placeholder')">
                <el-option v-for="item in roleOptions " :key="item.value" :label="item.label" :value="item.value">
                </el-option>
              </el-select>
            </el-form-item>
            <el-form-item>
              <div style="float: right">
                <el-button @click="onCancel()">{{ $t("commons.button.cancel") }}</el-button>
                <el-button type="primary" @click="onConfirm()">{{ $t("commons.button.confirm") }}
                </el-button>
              </div>
            </el-form-item>
          </el-form>
        </div>
      </el-col>
      <el-col :span="4"><br /></el-col>
    </el-row>
  </layout-content>
</template>

<script>
import LayoutContent from "@/components/layout/LayoutContent"
import { createGroup } from "@/api/groups"
import { listRoles } from "@/api/roles"
import Rules from "@/utils/rules"
import { getGroup, updateGroup } from "../../../../api/groups"

export default {
  name: "GroupCreate",
  props: ["name"],
  components: { LayoutContent },
  data() {
    return {
      group: {},
      loading: false,
      isSubmitGoing: false,
      roleOptions: [],
      rules: {
        name: [
          Rules.RequiredRule,
          Rules.CommonNameRule
        ],
        nickname: [
          Rules.RequiredRule
        ],
        roles: [
          Rules.RequiredRule,
        ],
      },
      form: {
        name: "",
        nickname: "",
        roles: [],
      },
    }
  },
  methods: {

    onConfirm() {
      if (this.isSubmitGoing) {
        return
      }
      let isFormReady = false
      this.$refs["form"].validate((valid) => {
        if (valid) {
          isFormReady = true
        }
      })
      if (!isFormReady) {
        return
      }
      this.isSubmitGoing = true
      this.loading = true
      console.log(this.name);
      if (this.name) {
        this.group.name = this.form.name
        this.group.nickname = this.form.nickname
        this.group.roles = this.form.roles
        console.log(this.group);
        updateGroup(this.name, this.group).then(() => {
          this.$message({
            type: "success",
            message: this.$t("commons.msg.update_success")
          })
          this.$router.push({ name: "Groups" })
        }).finally(() => {
          this.isSubmitGoing = false
          this.loading = false
        })
      } else {
        const req = {
          "apiVersion": "v1",
          "kind": "Group",
          "name": this.form.name,
          "roles": this.form.roles,
          "nickName": this.form.nickname,
        }
        createGroup(req).then(() => {
          this.$message({
            type: "success",
            message: this.$t("commons.msg.create_success")
          })
          this.$router.push({ name: "Groups" })
        }).finally(
          () => {
            this.isSubmitGoing = false
            this.loading = false
          }
        )
      }

    },

    onCancel() {
      this.$router.push({ name: "Groups" })
    },
    onCreated() {
      this.loading = true

      getGroup(this.name).then(data => {
        this.form.name = data.data.name
        this.form.nickname = data.data.nickName
        this.form.roles = data.data.roles
        this.user = data.data
      })
    },
  },



  created() {
    this.loading = true
    console.log(this);
    listRoles().then(d => {
      d.data.forEach(r => {
        this.roleOptions.push({
          label: r.name,
          value: r.name,
        })
      })
      this.loading = false
    })
    if (this.name) {
      this.onCreated()
    }
  }
}
</script>

<style scoped></style>
