<a name="detail-normal"></a>
#### 医生详情-普通

* url: `/doctor/detail`
* method: `get`
* 输入参数:  

| 参数名 | 类型 | 说明 | 例子 | 
| ----- | ---- | ---- | ---- | 
| userId | int |  用户id | 52347043|


* 测试数据:

```json
{
    "errcode": 0, 
    "errmsg": "操作成功", 
    "data": {
        "userId": "10001338",
        "name": "刘一钢",
        "isSurgeon": "1",
        "hospital": "福建医科大学附属协和医院",
        "hospitalId": "18016",
        "sectionId": "6",
        "titleId": "3",
        "districtId": "26",
        "areaId": "298",
        "creatorId": "235",
        "brokerId": "0",
        "workCertPhotoUrl": "",
        "workCertInfoUrl": "",
        "workCardNumber": "",
        "insertTime": "1411111596",
        "verifyTime": "1413287976",
        "cellphone": "11110001338",
        "sex": "0",
        "avatar": "http://7xp8vw.com1.z0.glb.clouddn.com/ef6244ca-2840-44dc-90ed-18a9a38d7b8b.jpg",
        "meAvatar": "http://7xp8vw.com1.z0.glb.clouddn.com/ef6244ca-2840-44dc-90ed-18a9a38d7b8b.jpg",
        "title": "医师",
        "sectionName": "消化内科",
        "hospitalLevel": "三级甲等",
        "inputDocOrInst": 2,
        "areaStr": "福建省,福州市",
        "lastActiveTime": "1472558161",
        "regPlatform": "未知",
        "creatorName": "卢群武",
        "brokerName": "无",
        "orderTotalPrice": 0,
        "orderCreatorCount": "0",
        "orderReceiveCount": "0"
     }
}
```

* 返回参数
 - userId : 用户id
 - cellphone : 电话
 - name : 姓名
 - sex : 性别（0-未设置，1-男，2-女）
 - titleId : 职称id
 - hospital : 医院
 - hospitalId : 医院id
 - sectionId : 科室id
 - sectionName : 科室名称
 - areaId : 地区id
 - areaStr : 地区（省、市、区）
 - avatar : 医联头像
 - meAvatar : me头像
 - hospitalLevel : 医院等级
 - isSurgeon : 是否为专家医生(1-普通医生, 2- 专家医生)
 - inputDocOrInst : 是否录入过专家医生和机构（0-否，为纯普通医生，1-录入过专家医生信息，2-录入过机构信息）
 - title : 职称
 - insertTime : 注册时间
 - verifyTime : 审核时间
 - lastActiveTime : 最后登录时间
 - regPlatform : 注册来源
 - creatorName : 推广人
 - brokerName : 经纪人
 - orderTotalPrice : 订单总额
 - orderCreatorCount : 发单数
 - orderReceiveCount : 接单数