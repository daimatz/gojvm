import java.util.ArrayList;

public class TreeTraversal {
    static class TreeNode {
        int val;
        TreeNode left, right;
        TreeNode(int val) { this.val = val; }
        TreeNode(int val, TreeNode left, TreeNode right) {
            this.val = val;
            this.left = left;
            this.right = right;
        }
    }

    // In-order traversal
    static void inorder(TreeNode n, ArrayList<Integer> result) {
        if (n == null) return;
        inorder(n.left, result);
        result.add(n.val);
        inorder(n.right, result);
    }

    // Tree depth
    static int depth(TreeNode n) {
        if (n == null) return 0;
        int ld = depth(n.left);
        int rd = depth(n.right);
        return 1 + (ld > rd ? ld : rd);
    }

    // Count nodes
    static int count(TreeNode n) {
        if (n == null) return 0;
        return 1 + count(n.left) + count(n.right);
    }

    public static void main(String[] args) {
        //       4
        //      / \
        //     2   6
        //    / \ / \
        //   1  3 5  7
        TreeNode root = new TreeNode(4,
            new TreeNode(2,
                new TreeNode(1),
                new TreeNode(3)),
            new TreeNode(6,
                new TreeNode(5),
                new TreeNode(7)));

        // In-order should give sorted output
        ArrayList<Integer> result = new ArrayList<>();
        inorder(root, result);
        for (int i = 0; i < result.size(); i++) {
            System.out.println(result.get(i));
        }
        // 1 2 3 4 5 6 7

        System.out.println(depth(root));  // 3
        System.out.println(count(root));  // 7
    }
}
