import { post, get, del, put } from "@/plugins/request"

const baseUrl = "/api/v1/groups"

export function searchGroups(pageNum, pageSize, conditions) {
    return post(`${baseUrl}/search?pageNum=${pageNum}&&pageSize=${pageSize}`, { conditions })
}

export function listGroups() {
    return get(`${baseUrl}`)
}

export function createGroup(group) {
    return post(`${baseUrl}`, group)
}

export function deleteGroup(name) {
    return del(`${baseUrl}/${name}`)
}

export function getGroup(name) {
    return get(`${baseUrl}/${name}`)
}

export function updateGroup(name, group) {
    return put(`${baseUrl}/${name}`, group)
}